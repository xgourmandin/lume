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
// Both layerID and tfWorkspaceID are stamped onto every parsed node so the
// frontend can filter and display provenance information.
type StateParser interface {
	Parse(ctx context.Context, stateData []byte, layerID, tfWorkspaceID string) (*domain.Organization, error)
}

// WorkspaceRepository defines the interface for persisting workspace hierarchy data.
type WorkspaceRepository interface {
	Save(ctx context.Context, workspace *domain.Workspace, hierarchy *domain.Organization) error
	GetByID(ctx context.Context, id string) (*domain.Workspace, *domain.Organization, error)
	ListWorkspaces(ctx context.Context) ([]*domain.Workspace, error)
	// SaveLayer persists the parsed Organization for a single (layerID, tfWorkspaceID) pair.
	// The document is keyed by "{layerID}--{tfWorkspaceID}" under the workspace's layers
	// sub-collection so that each Terraform workspace has its own isolated state snapshot.
	SaveLayer(ctx context.Context, workspaceID, layerID, tfWorkspaceID string, org *domain.Organization) error
	// GetMergedHierarchy fetches all layer/workspace state snapshots for the workspace
	// and returns a single Organization that is the result of merging them in order.
	GetMergedHierarchy(ctx context.Context, workspaceID string) (*domain.Organization, error)
}

// TofuService defines the orchestration logic for the Tofu/Terraform features.
type TofuService interface {
	// SyncWorkspace downloads, parses, and persists the state file identified by
	// (workspaceID, layerID, tfWorkspaceID). tfWorkspaceID is the Terraform workspace
	// name (e.g. "default", "prod", "staging") — distinct from the top-level workspaceID.
	SyncWorkspace(ctx context.Context, workspaceID, layerID, tfWorkspaceID, bucket, object string) (*domain.Organization, error)
}
