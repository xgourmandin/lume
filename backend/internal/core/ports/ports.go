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

// WorkspaceRepository defines the interface for persisting hierarchy data.
// There is a single implicit Landing Zone; no workspace identifier is needed.
type WorkspaceRepository interface {
	// SaveLayer persists the parsed Organization for a single (layerID, tfWorkspaceID) pair.
	SaveLayer(ctx context.Context, layerID, tfWorkspaceID string, org *domain.Organization) error
	// GetMergedHierarchy fetches all layer snapshots and returns a single merged Organization.
	GetMergedHierarchy(ctx context.Context) (*domain.Organization, error)
	// SaveDriftResult persists a drift report for a (layerID, tfWorkspaceID) pair.
	SaveDriftResult(ctx context.Context, layerID, tfWorkspaceID string, result *domain.DriftResult) error
	// GetDriftResult fetches the latest drift report for a (layerID, tfWorkspaceID) pair.
	GetDriftResult(ctx context.Context, layerID, tfWorkspaceID string) (*domain.DriftResult, error)
}

// PlanParser parses a Terraform/OpenTofu JSON plan file and derives a DriftResult.
// The status, counts, and timestamp are all computed by the implementation;
// callers do not supply them.
type PlanParser interface {
	ParseDrift(ctx context.Context, planData []byte) (*domain.DriftResult, error)
}

// TofuService defines the orchestration logic for the Tofu/Terraform features.
type TofuService interface {
	// SyncWorkspace downloads, parses, and persists the state file identified by
	// (layerID, tfWorkspaceID). The workspace is always the implicit single workspace.
	SyncWorkspace(ctx context.Context, layerID, tfWorkspaceID, bucket, object string) (*domain.Organization, error)
}
