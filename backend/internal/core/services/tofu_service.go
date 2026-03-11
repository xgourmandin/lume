package services

import (
	"context"
	"time"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

type TofuService struct {
	downloader ports.StateDownloader
	parser     ports.StateParser
	repo       ports.WorkspaceRepository
}

func NewTofuService(downloader ports.StateDownloader, parser ports.StateParser, repo ports.WorkspaceRepository) ports.TofuService {
	return &TofuService{
		downloader: downloader,
		parser:     parser,
		repo:       repo,
	}
}

func (s *TofuService) SyncWorkspace(ctx context.Context, workspaceID, layerID, bucket, object string) (*domain.Organization, error) {
	stateData, err := s.downloader.DownloadState(ctx, bucket, object)
	if err != nil {
		return nil, err
	}

	layerOrg, err := s.parser.Parse(ctx, stateData, layerID)
	if err != nil {
		return nil, err
	}

	// Persist the parsed org tree for this specific layer.
	if err := s.repo.SaveLayer(ctx, workspaceID, layerID, layerOrg); err != nil {
		return nil, err
	}

	// Build the merged view across all layers and update workspace metadata.
	merged, err := s.repo.GetMergedHierarchy(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	workspace := &domain.Workspace{
		ID:       workspaceID,
		LastSync: time.Now(),
		Status:   "clean",
		Layers: []domain.Layer{
			{
				ID:       layerID,
				Name:     layerID,
				LastSync: time.Now(),
				Status:   "clean",
			},
		},
	}

	if err := s.repo.Save(ctx, workspace, merged); err != nil {
		return nil, err
	}

	return merged, nil
}
