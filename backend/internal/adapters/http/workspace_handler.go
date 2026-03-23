package http

import (
	"time"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
	"github.com/mrshabel/mach"
)

type WorkspaceHandler struct {
	repo    ports.WorkspaceRepository
	service ports.TofuService
}

func NewWorkspaceHandler(repo ports.WorkspaceRepository, service ports.TofuService) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo, service: service}
}

func (h *WorkspaceHandler) Register(app *mach.App) {
	app.GET("/api/v1/workspaces", h.ListWorkspaces)
	app.GET("/api/v1/workspaces/{id}", h.GetWorkspace)
	app.GET("/api/v1/workspaces/{id}/drift/{layerId}/{tfWorkspace}", h.GetDriftResult)
	// CI/CD pipeline pushes drift results here
	app.POST("/api/v1/workspaces/{id}/drift/{layerId}/{tfWorkspace}", h.ReportDrift)
	app.GET("/api/v1/hierarchy/{id}", h.GetHierarchy)
	app.POST("/api/v1/hierarchy/sync", h.SyncHierarchy)
}

// ListWorkspaces returns all known workspace summaries (metadata only, no hierarchy).
func (h *WorkspaceHandler) ListWorkspaces(c *mach.Context) {
	workspaces, err := h.repo.ListWorkspaces(c.Request.Context())
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, workspaces)
}

// GetWorkspace returns the workspace metadata (layers + their TF workspaces) for a single workspace.
func (h *WorkspaceHandler) GetWorkspace(c *mach.Context) {
	id := c.Param("id")
	workspace, _, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, map[string]string{"error": "workspace not found"})
		return
	}
	c.JSON(200, workspace)
}

// GetHierarchy returns the fully merged GCP hierarchy for a workspace.
// Every node carries layer_id and workspace_id for client-side filtering.
func (h *WorkspaceHandler) GetHierarchy(c *mach.Context) {
	id := c.Param("id")
	_, hierarchy, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, map[string]string{"error": "workspace not found"})
		return
	}
	c.JSON(200, hierarchy)
}

// GetDriftResult returns the latest drift scan result for a (workspaceId, layerId, tfWorkspace) tuple.
// Corresponds to GET /api/v1/workspaces/{id}/drift/{layerId}/{tfWorkspace}
func (h *WorkspaceHandler) GetDriftResult(c *mach.Context) {
	workspaceID := c.Param("id")
	layerID := c.Param("layerId")
	tfWorkspace := c.Param("tfWorkspace")

	result, err := h.repo.GetDriftResult(c.Request.Context(), workspaceID, layerID, tfWorkspace)
	if err != nil {
		c.JSON(404, map[string]string{"error": "drift result not found"})
		return
	}
	c.JSON(200, result)
}

// ReportDrift receives a drift result posted by the CI/CD pipeline and persists it.
// The pipeline is responsible for running `tofu plan` and deriving add/change/destroy counts.
// Corresponds to POST /api/v1/workspaces/{id}/drift/{layerId}/{tfWorkspace}
//
// Request body:
//
//	{
//	  "status":        "clean" | "drifted" | "error",
//	  "add_count":     0,
//	  "change_count":  0,
//	  "destroy_count": 0,
//	  "scanned_at":    "2026-03-23T10:00:00Z",  // optional; defaults to server time
//	  "error_message": ""                         // optional
//	}
func (h *WorkspaceHandler) ReportDrift(c *mach.Context) {
	workspaceID := c.Param("id")
	layerID := c.Param("layerId")
	tfWorkspace := c.Param("tfWorkspace")

	var body struct {
		Status       string    `json:"status"`
		AddCount     int       `json:"add_count"`
		ChangeCount  int       `json:"change_count"`
		DestroyCount int       `json:"destroy_count"`
		ScannedAt    time.Time `json:"scanned_at"`
		ErrorMessage string    `json:"error_message"`
	}

	if err := c.DecodeJSON(&body); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	switch body.Status {
	case domain.DriftStatusClean, domain.DriftStatusDrifted, domain.DriftStatusError:
		// valid
	default:
		c.JSON(400, map[string]string{"error": "invalid status: must be clean, drifted, or error"})
		return
	}

	if body.ScannedAt.IsZero() {
		body.ScannedAt = time.Now().UTC()
	}

	result := &domain.DriftResult{
		Status:       body.Status,
		AddCount:     body.AddCount,
		ChangeCount:  body.ChangeCount,
		DestroyCount: body.DestroyCount,
		ScannedAt:    body.ScannedAt,
		ErrorMessage: body.ErrorMessage,
	}

	if err := h.repo.SaveDriftResult(c.Request.Context(), workspaceID, layerID, tfWorkspace, result); err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

func (h *WorkspaceHandler) SyncHierarchy(c *mach.Context) {
	var body struct {
		WorkspaceID   string `json:"workspace_id"`
		LayerID       string `json:"layer_id"`
		TFWorkspaceID string `json:"tf_workspace_id"` // Terraform workspace name, e.g. "prod"
		Bucket        string `json:"bucket"`
		Object        string `json:"object"`
	}

	if err := c.DecodeJSON(&body); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	if body.WorkspaceID == "" {
		body.WorkspaceID = "default"
	}
	if body.LayerID == "" {
		body.LayerID = "default"
	}
	if body.TFWorkspaceID == "" {
		body.TFWorkspaceID = "default"
	}

	org, err := h.service.SyncWorkspace(
		c.Request.Context(),
		body.WorkspaceID,
		body.LayerID,
		body.TFWorkspaceID,
		body.Bucket,
		body.Object,
	)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}
