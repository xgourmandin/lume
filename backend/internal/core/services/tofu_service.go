package services

import (
	"context"
	"fmt"
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
}

func NewTofuService(downloader ports.StateDownloader, parser ports.StateParser, repo ports.WorkspaceRepository) (ports.TofuService, error) {
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("GCS_BUCKET environment variable is not set")
	}
	return &TofuService{
		downloader: downloader,
		parser:     parser,
		repo:       repo,
		bucket:     bucket,
	}, nil
}

func (s *TofuService) SyncWorkspace(ctx context.Context, layerID, tfWorkspaceID, bucket, object string) (*domain.Organization, error) {
	stateData, err := s.downloader.DownloadState(ctx, bucket, object)
	if err != nil {
		return nil, err
	}

	layerOrg, err := s.parser.Parse(ctx, stateData, layerID, tfWorkspaceID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveLayer(ctx, layerID, tfWorkspaceID, layerOrg); err != nil {
		return nil, err
	}

	return s.repo.GetMergedHierarchy(ctx)
}

// SyncAllWorkspaces lists every .tfstate object in the configured GCS bucket
// and syncs each one. The layerID is derived from the object's directory and the
// tfWorkspaceID from the filename (without the .tfstate extension).
//
// Example:  "apps/prod.tfstate"  →  layerID="apps", tfWorkspaceID="prod"
//
//	"root.tfstate"       →  layerID="default", tfWorkspaceID="root"
func (s *TofuService) SyncAllWorkspaces(ctx context.Context) (*domain.SyncAllResult, error) {
	objects, err := s.downloader.ListStateObjects(ctx)
	if err != nil {
		return nil, err
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
			result.Failed++
			result.Errors = append(result.Errors, domain.SyncObjectError{
				Object: obj,
				Error:  err.Error(),
			})
			continue
		}
		result.Synced++
	}

	hierarchy, err := s.repo.GetMergedHierarchy(ctx)
	if err != nil {
		return nil, err
	}
	result.Hierarchy = hierarchy

	return result, nil
}
