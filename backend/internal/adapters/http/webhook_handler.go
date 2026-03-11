package http

import (
	"github.com/lume/backend/internal/core/ports"
	"github.com/mrshabel/mach"
)

type WebhookHandler struct {
	service ports.TofuService
}

func NewWebhookHandler(service ports.TofuService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (h *WebhookHandler) Register(app *mach.App) {
	app.POST("/api/v1/webhooks/git", h.HandleWebhook)
}

func (h *WebhookHandler) HandleWebhook(c *mach.Context) {
	var body struct {
		WorkspaceID string `json:"workspace_id"`
		Bucket      string `json:"bucket"`
		Object      string `json:"object"`
	}

	if err := c.DecodeJSON(&body); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	org, err := h.service.SyncWorkspace(c.Request.Context(), body.WorkspaceID, body.Bucket, body.Object)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}
