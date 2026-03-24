package services

import (
	"context"

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
