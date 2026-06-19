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
	if err := s.syncLayer(ctx, layerID, tfWorkspaceID, bucket, object); err != nil {
		return nil, err
	}

	hierarchy, err := s.repo.GetMergedHierarchy(ctx)
	if err != nil {
		return nil, fmt.Errorf("merge hierarchy: %w", err)
	}
	return hierarchy, nil
}

// syncLayer downloads, parses, and persists a single layer's state. It does NOT
// merge the full hierarchy: a single-object sync merges once on its own, and a
// bulk sync merges once after persisting every layer, so the merge stays O(1)
// instead of running per object. Errors are wrapped and returned for the caller
// to log once at the HTTP boundary.
func (s *TofuService) syncLayer(ctx context.Context, layerID, tfWorkspaceID, bucket, object string) error {
	log := s.logger.With("layer_id", layerID, "tf_workspace", tfWorkspaceID, "bucket", bucket, "object", object)
	log.DebugContext(ctx, "syncing layer")

	stateData, err := s.downloader.DownloadState(ctx, bucket, object)
	if err != nil {
		return fmt.Errorf("download state %q: %w", object, err)
	}
	log.DebugContext(ctx, "downloaded state", "bytes", len(stateData))

	layerOrg, err := s.parser.Parse(ctx, stateData, layerID, tfWorkspaceID)
	if err != nil {
		return fmt.Errorf("parse state %q: %w", object, err)
	}

	if err := s.repo.SaveLayer(ctx, layerID, tfWorkspaceID, layerOrg); err != nil {
		return fmt.Errorf("save layer %s/%s: %w", layerID, tfWorkspaceID, err)
	}

	log.InfoContext(ctx, "layer synced")
	return nil
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

		if err := s.syncLayer(ctx, layerID, tfWorkspaceID, s.bucket, obj); err != nil {
			// Record the per-object cause for the response; it is logged once
			// at the HTTP boundary rather than per object here.
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
		// layers exist to merge. Return the partial result so the boundary can
		// surface the per-object causes that led here.
		return result, fmt.Errorf("merge hierarchy after syncing %d/%d objects: %w", result.Synced, result.Synced+result.Failed, err)
	}
	result.Hierarchy = hierarchy

	return result, nil
}
