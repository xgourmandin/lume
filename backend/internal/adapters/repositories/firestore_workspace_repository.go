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
	// Project ID can be auto-detected from environment or explicitly passed if needed.
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
		HierarchyCache *domain.Organization `firestore:"hierarchy_cache"`
	}

	if err := doc.DataTo(&data); err != nil {
		return nil, nil, err
	}

	workspace := &domain.Workspace{
		ID:       data.ID,
		LastSync: data.LastSync,
		Status:   data.Status,
	}

	return workspace, data.HierarchyCache, nil
}
