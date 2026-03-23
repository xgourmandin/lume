package repositories

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

type FirestoreWorkspaceRepository struct {
	client *firestore.Client
}

func NewFirestoreWorkspaceRepository() (ports.WorkspaceRepository, error) {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, firestore.DetectProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}
	return &FirestoreWorkspaceRepository{client: client}, nil
}

func (r *FirestoreWorkspaceRepository) Save(ctx context.Context, workspace *domain.Workspace, hierarchy *domain.Organization) error {
	docRef := r.client.Collection("workspaces").Doc(workspace.ID)
	_, err := docRef.Set(ctx, map[string]interface{}{
		"id":              workspace.ID,
		"last_sync":       time.Now(),
		"status":          workspace.Status,
		"layers":          workspace.Layers,
		"hierarchy_cache": hierarchy,
	})
	return err
}

func (r *FirestoreWorkspaceRepository) GetByID(ctx context.Context, id string) (*domain.Workspace, *domain.Organization, error) {
	doc, err := r.client.Collection("workspaces").Doc(id).Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	var data struct {
		ID             string               `firestore:"id"`
		LastSync       time.Time            `firestore:"last_sync"`
		Status         string               `firestore:"status"`
		Layers         []domain.Layer       `firestore:"layers"`
		HierarchyCache *domain.Organization `firestore:"hierarchy_cache"`
	}

	if err := doc.DataTo(&data); err != nil {
		return nil, nil, err
	}

	workspace := &domain.Workspace{
		ID:       data.ID,
		LastSync: data.LastSync,
		Status:   data.Status,
		Layers:   data.Layers,
	}

	return workspace, data.HierarchyCache, nil
}

// ListWorkspaces returns summary metadata for all workspaces (no hierarchy cache).
func (r *FirestoreWorkspaceRepository) ListWorkspaces(ctx context.Context) ([]*domain.Workspace, error) {
	docs, err := r.client.Collection("workspaces").Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	workspaces := make([]*domain.Workspace, 0, len(docs))
	for _, doc := range docs {
		var data struct {
			ID       string         `firestore:"id"`
			LastSync time.Time      `firestore:"last_sync"`
			Status   string         `firestore:"status"`
			Layers   []domain.Layer `firestore:"layers"`
		}
		if err := doc.DataTo(&data); err != nil {
			return nil, fmt.Errorf("failed to decode workspace %s: %w", doc.Ref.ID, err)
		}
		workspaces = append(workspaces, &domain.Workspace{
			ID:       data.ID,
			LastSync: data.LastSync,
			Status:   data.Status,
			Layers:   data.Layers,
		})
	}
	return workspaces, nil
}

// SaveLayer persists the parsed Organization for a single (layerID, tfWorkspaceID) pair.
// The document is keyed as "{layerID}--{tfWorkspaceID}" so that each Terraform workspace
// gets its own isolated state snapshot while remaining queryable as a flat collection.
func (r *FirestoreWorkspaceRepository) SaveLayer(ctx context.Context, workspaceID, layerID, tfWorkspaceID string, org *domain.Organization) error {
	docID := layerID + "--" + tfWorkspaceID
	docRef := r.client.
		Collection("workspaces").Doc(workspaceID).
		Collection("layers").Doc(docID)

	_, err := docRef.Set(ctx, map[string]interface{}{
		"layer_id":     layerID,
		"workspace_id": tfWorkspaceID,
		"last_sync":    time.Now(),
		"hierarchy":    org,
	})
	return err
}

// GetMergedHierarchy fetches every (layerID, tfWorkspaceID) state snapshot for the
// workspace and merges them into a single Organization using the domain Merge logic.
func (r *FirestoreWorkspaceRepository) GetMergedHierarchy(ctx context.Context, workspaceID string) (*domain.Organization, error) {
	docs, err := r.client.
		Collection("workspaces").Doc(workspaceID).
		Collection("layers").
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch layers for workspace %s: %w", workspaceID, err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no layers found for workspace %s", workspaceID)
	}

	var merged *domain.Organization
	for _, doc := range docs {
		var data struct {
			Hierarchy *domain.Organization `firestore:"hierarchy"`
		}
		if err := doc.DataTo(&data); err != nil {
			return nil, fmt.Errorf("failed to decode layer %s: %w", doc.Ref.ID, err)
		}
		if data.Hierarchy == nil {
			continue
		}
		if merged == nil {
			merged = data.Hierarchy
		} else {
			merged.Merge(data.Hierarchy)
		}
	}

	if merged == nil {
		return nil, fmt.Errorf("no valid hierarchy found in layers for workspace %s", workspaceID)
	}
	return merged, nil
}

// SaveDriftResult persists a DriftResult in the drift_results sub-collection and
// updates the corresponding TerraformWorkspace status on the parent workspace document.
func (r *FirestoreWorkspaceRepository) SaveDriftResult(ctx context.Context, workspaceID, layerID, tfWorkspaceID string, result *domain.DriftResult) error {
	docID := layerID + "--" + tfWorkspaceID

	// Write the drift result document.
	driftRef := r.client.
		Collection("workspaces").Doc(workspaceID).
		Collection("drift_results").Doc(docID)

	if _, err := driftRef.Set(ctx, map[string]interface{}{
		"layer_id":      layerID,
		"workspace_id":  tfWorkspaceID,
		"status":        result.Status,
		"add_count":     result.AddCount,
		"change_count":  result.ChangeCount,
		"destroy_count": result.DestroyCount,
		"scanned_at":    result.ScannedAt,
		"error_message": result.ErrorMessage,
	}); err != nil {
		return fmt.Errorf("failed to write drift result: %w", err)
	}

	// Best-effort: update the parent workspace's layer/workspace status.
	workspace, _, err := r.GetByID(ctx, workspaceID)
	if err != nil {
		// Workspace may not exist yet (state sync hasn't run); drift result is saved, skip metadata update.
		return nil
	}

	now := time.Now()
	for i := range workspace.Layers {
		if workspace.Layers[i].ID != layerID {
			continue
		}
		workspace.Layers[i].Status = result.Status
		workspace.Layers[i].LastSync = now
		for j := range workspace.Layers[i].Workspaces {
			if workspace.Layers[i].Workspaces[j].ID == tfWorkspaceID {
				workspace.Layers[i].Workspaces[j].Status = result.Status
				workspace.Layers[i].Workspaces[j].LastSync = now
				break
			}
		}
		break
	}
	workspace.Status = result.Status
	workspace.LastSync = now

	_, err = r.client.Collection("workspaces").Doc(workspaceID).Update(ctx, []firestore.Update{
		{Path: "layers", Value: workspace.Layers},
		{Path: "status", Value: workspace.Status},
		{Path: "last_sync", Value: workspace.LastSync},
	})
	return err
}

// GetDriftResult fetches the latest drift scan result for a single
// (workspaceID, layerID, tfWorkspaceID) tuple from the drift_results sub-collection.
func (r *FirestoreWorkspaceRepository) GetDriftResult(ctx context.Context, workspaceID, layerID, tfWorkspaceID string) (*domain.DriftResult, error) {
	docID := layerID + "--" + tfWorkspaceID
	doc, err := r.client.
		Collection("workspaces").Doc(workspaceID).
		Collection("drift_results").Doc(docID).
		Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("drift result not found for %s/%s/%s: %w", workspaceID, layerID, tfWorkspaceID, err)
	}

	var data struct {
		Status       string    `firestore:"status"`
		AddCount     int       `firestore:"add_count"`
		ChangeCount  int       `firestore:"change_count"`
		DestroyCount int       `firestore:"destroy_count"`
		ScannedAt    time.Time `firestore:"scanned_at"`
		ErrorMessage string    `firestore:"error_message"`
	}
	if err := doc.DataTo(&data); err != nil {
		return nil, fmt.Errorf("failed to decode drift result: %w", err)
	}

	return &domain.DriftResult{
		Status:       data.Status,
		AddCount:     data.AddCount,
		ChangeCount:  data.ChangeCount,
		DestroyCount: data.DestroyCount,
		ScannedAt:    data.ScannedAt,
		ErrorMessage: data.ErrorMessage,
	}, nil
}
