package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

type TofuService struct {
	downloader ports.StateDownloader
	parser     ports.StateParser
	repo       ports.WorkspaceRepository
	bucket     string
	logger     *slog.Logger
}

func NewTofuService(downloader ports.StateDownloader, parser ports.StateParser, repo ports.WorkspaceRepository, logger *slog.Logger) (ports.TofuService, error) {
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("GCS_BUCKET environment variable is not set")
	}
	return &TofuService{
		downloader: downloader,
		parser:     parser,
		repo:       repo,
		bucket:     bucket,
		logger:     logger.With("component", "tofu_service"),
	}, nil
}

func (s *TofuService) SyncWorkspace(ctx context.Context, layerID, tfWorkspaceID, bucket, object string) (*domain.Organization, error) {
	log := s.logger.With("layer_id", layerID, "tf_workspace", tfWorkspaceID, "bucket", bucket, "object", object)
	log.DebugContext(ctx, "syncing workspace")

	stateData, err := s.downloader.DownloadState(ctx, bucket, object)
	if err != nil {
		log.ErrorContext(ctx, "failed to download state", "error", err)
		return nil, fmt.Errorf("download state %q: %w", object, err)
	}
	log.DebugContext(ctx, "downloaded state", "bytes", len(stateData))

	layerOrg, err := s.parser.Parse(ctx, stateData, layerID, tfWorkspaceID)
	if err != nil {
		log.ErrorContext(ctx, "failed to parse state", "error", err)
		return nil, fmt.Errorf("parse state %q: %w", object, err)
	}

	if err := s.repo.SaveLayer(ctx, layerID, tfWorkspaceID, layerOrg); err != nil {
		log.ErrorContext(ctx, "failed to save layer", "error", err)
		return nil, fmt.Errorf("save layer %s/%s: %w", layerID, tfWorkspaceID, err)
	}

	hierarchy, err := s.repo.GetMergedHierarchy(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to merge hierarchy", "error", err)
		return nil, fmt.Errorf("merge hierarchy: %w", err)
	}

	log.InfoContext(ctx, "workspace synced")
	return hierarchy, nil
}

// SyncAllWorkspaces lists every .tfstate object in the configured GCS bucket
// and syncs each one. The layerID is derived from the object's directory and the
// tfWorkspaceID from the filename (without the .tfstate extension).
//
// Example:  "apps/prod.tfstate"  →  layerID="apps", tfWorkspaceID="prod"
//
//	"root.tfstate"       →  layerID="default", tfWorkspaceID="root"
func (s *TofuService) SyncAllWorkspaces(ctx context.Context) (*domain.SyncAllResult, error) {
	log := s.logger.With("operation", "sync_all", "bucket", s.bucket)
	log.InfoContext(ctx, "starting bulk sync")

	objects, err := s.downloader.ListStateObjects(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list state objects", "error", err)
		return nil, fmt.Errorf("list state objects in bucket %q: %w", s.bucket, err)
	}

	if len(objects) == 0 {
		log.WarnContext(ctx, "no .tfstate objects found in bucket")
	} else {
		log.InfoContext(ctx, "found state objects", "count", len(objects))
	}

	result := &domain.SyncAllResult{}

	for _, obj := range objects {
		dir := path.Dir(obj)
		layerID := dir
		if dir == "." || dir == "" {
			layerID = "default"
		}

		base := path.Base(obj)
		tfWorkspaceID := strings.TrimSuffix(base, ".tfstate")

		if _, err := s.SyncWorkspace(ctx, layerID, tfWorkspaceID, s.bucket, obj); err != nil {
			// SyncWorkspace already logged the underlying cause; record it for the response.
			result.Failed++
			result.Errors = append(result.Errors, domain.SyncObjectError{
				Object: obj,
				Error:  err.Error(),
			})
			continue
		}
		result.Synced++
	}

	log.InfoContext(ctx, "bulk sync finished", "synced", result.Synced, "failed", result.Failed)

	hierarchy, err := s.repo.GetMergedHierarchy(ctx)
	if err != nil {
		// This is the common opaque-500 case: every object failed to sync, so no
		// layers exist to merge. Surface the per-object causes that led here.
		log.ErrorContext(ctx, "failed to merge hierarchy after bulk sync",
			"error", err,
			"synced", result.Synced,
			"failed", result.Failed,
			"object_errors", result.Errors,
		)
		return nil, fmt.Errorf("merge hierarchy after syncing %d/%d objects: %w", result.Synced, result.Synced+result.Failed, err)
	}
	result.Hierarchy = hierarchy

	return result, nil
}
