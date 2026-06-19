package repositories

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

// Firestore layout (flat, single-workspace):
//
//	layers/{layerID--tfWorkspaceID}        → hierarchy snapshot per (layer, tf-workspace)
//	drift_results/{layerID--tfWorkspaceID} → latest drift report per (layer, tf-workspace)

type FirestoreWorkspaceRepository struct {
	client *firestore.Client
	logger *slog.Logger
}

func NewFirestoreWorkspaceRepository(logger *slog.Logger) (ports.WorkspaceRepository, error) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = firestore.DetectProjectID
	}
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}
	return &FirestoreWorkspaceRepository{client: client, logger: logger.With("component", "firestore_repo")}, nil
}

// SaveLayer persists the parsed Organization for a single (layerID, tfWorkspaceID) pair.
// The document is keyed as "{layerID}--{tfWorkspaceID}".
func (r *FirestoreWorkspaceRepository) SaveLayer(ctx context.Context, layerID, tfWorkspaceID string, org *domain.Organization) error {
	docID := layerID + "--" + tfWorkspaceID
	_, err := r.client.Collection("layers").Doc(docID).Set(ctx, map[string]interface{}{
		"layer_id":     layerID,
		"tf_workspace": tfWorkspaceID,
		"last_sync":    time.Now(),
		"hierarchy":    org,
	})
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to save layer", "doc_id", docID, "error", err)
		return fmt.Errorf("save layer doc %q: %w", docID, err)
	}
	r.logger.DebugContext(ctx, "saved layer", "doc_id", docID)
	return nil
}

// GetMergedHierarchy fetches every layer snapshot and deep-merges them into a
// single Organization using the domain Merge logic.
func (r *FirestoreWorkspaceRepository) GetMergedHierarchy(ctx context.Context) (*domain.Organization, error) {
	docs, err := r.client.Collection("layers").Documents(ctx).GetAll()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to fetch layers", "error", err)
		return nil, fmt.Errorf("failed to fetch layers: %w", err)
	}
	if len(docs) == 0 {
		r.logger.WarnContext(ctx, "no layers found; nothing has been synced yet")
		return nil, fmt.Errorf("no layers found")
	}

	var merged *domain.Organization
	for _, doc := range docs {
		var data struct {
			Hierarchy *domain.Organization `firestore:"hierarchy"`
		}
		if err := doc.DataTo(&data); err != nil {
			r.logger.ErrorContext(ctx, "failed to decode layer", "doc_id", doc.Ref.ID, "error", err)
			return nil, fmt.Errorf("failed to decode layer %s: %w", doc.Ref.ID, err)
		}
		if data.Hierarchy == nil {
			r.logger.WarnContext(ctx, "layer has no hierarchy payload; skipping", "doc_id", doc.Ref.ID)
			continue
		}
		if merged == nil {
			merged = data.Hierarchy
		} else {
			merged.Merge(data.Hierarchy)
		}
	}

	if merged == nil {
		r.logger.WarnContext(ctx, "no valid hierarchy found in layers", "layer_count", len(docs))
		return nil, fmt.Errorf("no valid hierarchy found in layers")
	}
	r.logger.DebugContext(ctx, "merged hierarchy", "layer_count", len(docs))
	return merged, nil
}

// SaveDriftResult persists a DriftResult for a single (layerID, tfWorkspaceID) pair.
func (r *FirestoreWorkspaceRepository) SaveDriftResult(ctx context.Context, layerID, tfWorkspaceID string, result *domain.DriftResult) error {
	docID := layerID + "--" + tfWorkspaceID
	_, err := r.client.Collection("drift_results").Doc(docID).Set(ctx, map[string]interface{}{
		"layer_id":      layerID,
		"tf_workspace":  tfWorkspaceID,
		"status":        result.Status,
		"add_count":     result.AddCount,
		"change_count":  result.ChangeCount,
		"destroy_count": result.DestroyCount,
		"scanned_at":    result.ScannedAt,
		"error_message": result.ErrorMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to write drift result: %w", err)
	}
	return nil
}

// GetDriftResult fetches the latest drift scan result for a (layerID, tfWorkspaceID) pair.
func (r *FirestoreWorkspaceRepository) GetDriftResult(ctx context.Context, layerID, tfWorkspaceID string) (*domain.DriftResult, error) {
	docID := layerID + "--" + tfWorkspaceID
	doc, err := r.client.Collection("drift_results").Doc(docID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("drift result not found for %s/%s: %w", layerID, tfWorkspaceID, err)
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
