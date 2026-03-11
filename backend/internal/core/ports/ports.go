package ports

import (
	"context"

	"github.com/lume/backend/internal/core/domain"
)

// StateDownloader defines the interface for fetching the .tfstate file.
type StateDownloader interface {
	DownloadState(ctx context.Context, bucketName, objectName string) ([]byte, error)
}

// StateParser defines the interface for parsing the .tfstate JSON into a hierarchy.
type StateParser interface {
	Parse(ctx context.Context, stateData []byte) (*domain.Organization, error)
}

// WorkspaceRepository defines the interface for persisting workspace hierarchy data.
type WorkspaceRepository interface {
	Save(ctx context.Context, workspace *domain.Workspace, hierarchy *domain.Organization) error
	GetByID(ctx context.Context, id string) (*domain.Workspace, *domain.Organization, error)
}

// TofuService defines the orchestration logic for the Tofu/Terraform features.
type TofuService interface {
	SyncWorkspace(ctx context.Context, workspaceID, bucket, object string) (*domain.Organization, error)
}

// GitProvider defines the interface for interacting with VCS systems.
type GitProvider interface {
	CreatePullRequest(ctx context.Context, repo, branch, title, body string, files map[string]string) (string, error)
}
