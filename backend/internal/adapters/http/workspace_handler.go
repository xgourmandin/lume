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
	app.GET("/api/v1/hierarchy/{id}", h.GetHierarchy)
	app.POST("/api/v1/hierarchy/sync", h.SyncHierarchy)
}

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
		WorkspaceID string `json:"workspace_id"`
		Bucket      string `json:"bucket"`
		Object      string `json:"object"`
	}

	if err := c.DecodeJSON(&body); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	// Default to "default" workspace if not provided
	if body.WorkspaceID == "" {
		body.WorkspaceID = "default"
	}

	org, err := h.service.SyncWorkspace(c.Request.Context(), body.WorkspaceID, body.Bucket, body.Object)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}
