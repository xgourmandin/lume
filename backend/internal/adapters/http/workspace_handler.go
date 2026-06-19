package http

import (
	"io"
	"log/slog"

	"github.com/lume/backend/internal/core/ports"
	"github.com/mrshabel/mach"
)

type WorkspaceHandler struct {
	repo       ports.WorkspaceRepository
	service    ports.TofuService
	planParser ports.PlanParser
	logger     *slog.Logger
}

func NewWorkspaceHandler(repo ports.WorkspaceRepository, service ports.TofuService, planParser ports.PlanParser, logger *slog.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo, service: service, planParser: planParser, logger: logger.With("component", "workspace_handler")}
}

func (h *WorkspaceHandler) Register(app *mach.App) {
	app.GET("/api/v1/hierarchy", h.GetHierarchy)
	app.POST("/api/v1/hierarchy/sync", h.SyncHierarchy)
	app.POST("/api/v1/hierarchy/sync-all", h.SyncAllHierarchy)
	app.GET("/api/v1/drift/{layerId}/{tfWorkspace}", h.GetDriftResult)
	app.POST("/api/v1/drift/{layerId}/{tfWorkspace}", h.ReportDrift)
}

// GetHierarchy returns the fully merged GCP hierarchy.
func (h *WorkspaceHandler) GetHierarchy(c *mach.Context) {
	hierarchy, err := h.repo.GetMergedHierarchy(c.Request.Context())
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "get hierarchy failed", "error", err)
		c.JSON(404, map[string]string{"error": "hierarchy not found"})
		return
	}
	c.JSON(200, hierarchy)
}

// GetDriftResult returns the latest drift scan result for a (layerId, tfWorkspace) tuple.
// Corresponds to GET /api/v1/drift/{layerId}/{tfWorkspace}
func (h *WorkspaceHandler) GetDriftResult(c *mach.Context) {
	layerID := c.Param("layerId")
	tfWorkspace := c.Param("tfWorkspace")

	result, err := h.repo.GetDriftResult(c.Request.Context(), layerID, tfWorkspace)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "get drift result failed",
			"layer_id", layerID, "tf_workspace", tfWorkspace, "error", err)
		c.JSON(404, map[string]string{"error": "drift result not found"})
		return
	}
	c.JSON(200, result)
}

// ReportDrift receives a Terraform/OpenTofu JSON plan file from the CI/CD pipeline,
// derives the drift status and resource-change counts, and persists the result.
//
// The request must be multipart/form-data with a single field named "plan"
// whose value is the JSON output of `tofu show -json <planfile>`.
//
// Corresponds to POST /api/v1/drift/{layerId}/{tfWorkspace}
func (h *WorkspaceHandler) ReportDrift(c *mach.Context) {
	ctx := c.Request.Context()
	layerID := c.Param("layerId")
	tfWorkspace := c.Param("tfWorkspace")
	log := h.logger.With("layer_id", layerID, "tf_workspace", tfWorkspace)

	// Parse up to 32 MB of multipart data.
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		log.WarnContext(ctx, "invalid multipart form", "error", err)
		c.JSON(400, map[string]string{"error": "expected multipart/form-data request"})
		return
	}

	file, _, err := c.Request.FormFile("plan")
	if err != nil {
		log.WarnContext(ctx, "missing 'plan' file field", "error", err)
		c.JSON(400, map[string]string{"error": "missing 'plan' file field"})
		return
	}
	defer file.Close()

	planData, err := io.ReadAll(file)
	if err != nil {
		log.ErrorContext(ctx, "failed to read plan file", "error", err)
		c.JSON(500, map[string]string{"error": "failed to read plan file"})
		return
	}

	result, err := h.planParser.ParseDrift(ctx, planData)
	if err != nil {
		log.WarnContext(ctx, "failed to parse drift plan", "plan_bytes", len(planData), "error", err)
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	if err := h.repo.SaveDriftResult(ctx, layerID, tfWorkspace, result); err != nil {
		log.ErrorContext(ctx, "failed to save drift result", "error", err)
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	log.InfoContext(ctx, "drift reported", "status", result.Status,
		"add", result.AddCount, "change", result.ChangeCount, "destroy", result.DestroyCount)
	c.JSON(200, result)
}

// SyncHierarchy triggers a state file download and hierarchy merge from GCS.
// Corresponds to POST /api/v1/hierarchy/sync
//
// Request body:
//
//	{
//	  "layer_id":       "apps",          // optional; defaults to "default"
//	  "tf_workspace_id":"prod",          // optional; defaults to "default"
//	  "bucket":         "my-tf-states",
//	  "object":         "apps/prod.tfstate"
//	}
func (h *WorkspaceHandler) SyncHierarchy(c *mach.Context) {
	var body struct {
		LayerID       string `json:"layer_id"`
		TFWorkspaceID string `json:"tf_workspace_id"`
		Bucket        string `json:"bucket"`
		Object        string `json:"object"`
	}

	ctx := c.Request.Context()
	if err := c.DecodeJSON(&body); err != nil {
		h.logger.WarnContext(ctx, "invalid sync request body", "error", err)
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	if body.LayerID == "" {
		body.LayerID = "default"
	}
	if body.TFWorkspaceID == "" {
		body.TFWorkspaceID = "default"
	}

	org, err := h.service.SyncWorkspace(
		ctx,
		body.LayerID,
		body.TFWorkspaceID,
		body.Bucket,
		body.Object,
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "sync request failed",
			"layer_id", body.LayerID, "tf_workspace", body.TFWorkspaceID, "bucket", body.Bucket, "object", body.Object, "error", err)
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}

// SyncAllHierarchy lists every .tfstate file in the configured GCS bucket and
// syncs them all in one shot. The layerID and tfWorkspaceID for each object are
// inferred from its path: the directory segment becomes the layerID and the
// filename (without the .tfstate extension) becomes the tfWorkspaceID.
//
// Corresponds to POST /api/v1/hierarchy/sync-all
//
// Response body:
//
//	{
//	  "synced":    5,
//	  "failed":    1,
//	  "errors":    [{"object": "apps/broken.tfstate", "error": "..."}],
//	  "hierarchy": { ... }
//	}
func (h *WorkspaceHandler) SyncAllHierarchy(c *mach.Context) {
	ctx := c.Request.Context()
	result, err := h.service.SyncAllWorkspaces(ctx)
	if err != nil {
		// result is non-nil when objects were synced but the final merge failed;
		// surface the per-object causes that led to the opaque 500.
		attrs := []any{"error", err}
		if result != nil {
			attrs = append(attrs, "synced", result.Synced, "failed", result.Failed, "object_errors", result.Errors)
		}
		h.logger.ErrorContext(ctx, "sync-all request failed", attrs...)
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	// Even on overall success, individual objects may have failed to sync.
	if result.Failed > 0 {
		h.logger.WarnContext(ctx, "sync-all completed with per-object failures",
			"synced", result.Synced, "failed", result.Failed, "errors", result.Errors)
	}

	c.JSON(200, result)
}
