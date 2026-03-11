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

// SaveLayer persists the parsed Organization for a single state layer under
// workspaces/{workspaceID}/layers/{layerID}.
func (r *FirestoreWorkspaceRepository) SaveLayer(ctx context.Context, workspaceID, layerID string, org *domain.Organization) error {
	docRef := r.client.
		Collection("workspaces").Doc(workspaceID).
		Collection("layers").Doc(layerID)

	_, err := docRef.Set(ctx, map[string]interface{}{
		"id":        layerID,
		"last_sync": time.Now(),
		"hierarchy": org,
	})
	return err
}

// GetMergedHierarchy fetches every layer document for the workspace and
// merges them into a single Organization using the domain's own Merge logic.
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
