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

func (s *TofuService) SyncWorkspace(ctx context.Context, workspaceID, bucket, object string) (*domain.Organization, error) {
	stateData, err := s.downloader.DownloadState(ctx, bucket, object)
	if err != nil {
		return nil, err
	}

	org, err := s.parser.Parse(ctx, stateData)
	if err != nil {
		return nil, err
	}

	workspace := &domain.Workspace{
		ID:     workspaceID,
		Status: "clean",
	}

	if err := s.repo.Save(ctx, workspace, org); err != nil {
		return nil, err
	}

	return org, nil
}
