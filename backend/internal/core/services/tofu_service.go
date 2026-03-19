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

func (s *TofuService) SyncWorkspace(ctx context.Context, workspaceID, layerID, tfWorkspaceID, bucket, object string) (*domain.Organization, error) {
	stateData, err := s.downloader.DownloadState(ctx, bucket, object)
	if err != nil {
		return nil, err
	}

	// Parse the state, stamping every node with both layerID and tfWorkspaceID.
	layerOrg, err := s.parser.Parse(ctx, stateData, layerID, tfWorkspaceID)
	if err != nil {
		return nil, err
	}

	// Persist this state snapshot keyed by (layerID, tfWorkspaceID).
	if err := s.repo.SaveLayer(ctx, workspaceID, layerID, tfWorkspaceID, layerOrg); err != nil {
		return nil, err
	}

	// Build the merged view across all layers/workspaces.
	merged, err := s.repo.GetMergedHierarchy(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Load existing workspace metadata so we preserve previously synced layers.
	workspace, _, _ := s.repo.GetByID(ctx, workspaceID) // best-effort; nil on first sync
	if workspace == nil {
		workspace = &domain.Workspace{ID: workspaceID}
	}
	workspace.LastSync = time.Now()
	workspace.Status = "clean" // TODO: derive from plan diff

	// Upsert the (layerID, tfWorkspaceID) entry in the workspace metadata.
	upsertLayerWorkspace(workspace, layerID, tfWorkspaceID)

	if err := s.repo.Save(ctx, workspace, merged); err != nil {
		return nil, err
	}

	return merged, nil
}

// upsertLayerWorkspace ensures the given layerID exists in ws.Layers and that
// tfWorkspaceID appears in that layer's Workspaces slice, updating timestamps.
func upsertLayerWorkspace(ws *domain.Workspace, layerID, tfWorkspaceID string) {
	now := time.Now()
	for i := range ws.Layers {
		if ws.Layers[i].ID == layerID {
			ws.Layers[i].LastSync = now
			upsertTFWorkspace(&ws.Layers[i], tfWorkspaceID, now)
			return
		}
	}
	// Layer not found — create it with the new TF workspace.
	ws.Layers = append(ws.Layers, domain.Layer{
		ID:       layerID,
		Name:     layerID,
		LastSync: now,
		Status:   "clean",
		Workspaces: []domain.TerraformWorkspace{
			{ID: tfWorkspaceID, LayerID: layerID, Status: "clean", LastSync: now},
		},
	})
}

// upsertTFWorkspace adds or refreshes a TerraformWorkspace entry within a layer.
func upsertTFWorkspace(layer *domain.Layer, tfWorkspaceID string, now time.Time) {
	for i := range layer.Workspaces {
		if layer.Workspaces[i].ID == tfWorkspaceID {
			layer.Workspaces[i].LastSync = now
			layer.Workspaces[i].Status = "clean"
			return
		}
	}
	layer.Workspaces = append(layer.Workspaces, domain.TerraformWorkspace{
		ID:       tfWorkspaceID,
		LayerID:  layer.ID,
		Status:   "clean",
		LastSync: now,
	})
}
