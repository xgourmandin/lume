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
	Parse(ctx context.Context, stateData []byte, layerID string) (*domain.Organization, error)
}

// WorkspaceRepository defines the interface for persisting workspace hierarchy data.
type WorkspaceRepository interface {
	Save(ctx context.Context, workspace *domain.Workspace, hierarchy *domain.Organization) error
	GetByID(ctx context.Context, id string) (*domain.Workspace, *domain.Organization, error)
	// SaveLayer persists the parsed Organization for a single state layer.
	SaveLayer(ctx context.Context, workspaceID, layerID string, org *domain.Organization) error
	// GetMergedHierarchy fetches all layers for the workspace and returns a
	// single Organization that is the result of merging them in order.
	GetMergedHierarchy(ctx context.Context, workspaceID string) (*domain.Organization, error)
}

// TofuService defines the orchestration logic for the Tofu/Terraform features.
type TofuService interface {
	SyncWorkspace(ctx context.Context, workspaceID, layerID, bucket, object string) (*domain.Organization, error)
}
