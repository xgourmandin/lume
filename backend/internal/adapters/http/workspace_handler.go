package http

import (
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
