package http

import (
	"encoding/base64"
	"encoding/json"
	"strings"

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
	app.POST("/api/v1/webhooks/gcs", h.HandleGCSNotification)
}

func (h *WebhookHandler) HandleWebhook(c *mach.Context) {
	var body struct {
		WorkspaceID string `json:"workspace_id"`
		LayerID     string `json:"layer_id"`
		Bucket      string `json:"bucket"`
		Object      string `json:"object"`
	}

	if err := c.DecodeJSON(&body); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	if body.LayerID == "" {
		body.LayerID = "default"
	}

	org, err := h.service.SyncWorkspace(c.Request.Context(), body.WorkspaceID, body.LayerID, body.Bucket, body.Object)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}

// pubSubMessage is the outer Pub/Sub push envelope.
type pubSubMessage struct {
	Message struct {
		Data string `json:"data"` // base64-encoded GCS notification JSON
	} `json:"message"`
}

// gcsNotification is the inner JSON payload encoded in message.data.
// See: https://cloud.google.com/storage/docs/pubsub-notifications#format
type gcsNotification struct {
	Bucket    string `json:"bucket"`
	Name      string `json:"name"`      // full object path, e.g. prod/network/default.tfstate
	EventType string `json:"eventType"` // e.g. OBJECT_FINALIZE
}

// HandleGCSNotification handles Pub/Sub push messages sent by GCS object change notifications.
// Only OBJECT_FINALIZE events for .tfstate files trigger a workspace sync; all other events
// are acknowledged with 200 OK and ignored.
//
// GCS object path convention: {workspaceID}/{layerID}/default.tfstate
//   - "prod/network/default.tfstate"  → workspace="prod",    layer="network"
//   - "prod/default.tfstate"          → workspace="prod",    layer="default"
//   - "default.tfstate"               → workspace="default", layer="default"
func (h *WebhookHandler) HandleGCSNotification(c *mach.Context) {
	var envelope pubSubMessage
	if err := c.DecodeJSON(&envelope); err != nil {
		c.JSON(400, map[string]string{"error": "invalid pubsub envelope"})
		return
	}

	rawData, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		c.JSON(400, map[string]string{"error": "invalid base64 in message.data"})
		return
	}

	var notification gcsNotification
	if err := json.Unmarshal(rawData, &notification); err != nil {
		c.JSON(400, map[string]string{"error": "invalid gcs notification payload"})
		return
	}

	// Only process object creation/upload events for state files.
	if notification.EventType != "OBJECT_FINALIZE" || !strings.HasSuffix(notification.Name, ".tfstate") {
		c.JSON(200, map[string]string{"status": "ignored"})
		return
	}

	workspaceID, layerID := parseObjectPath(notification.Name)

	org, err := h.service.SyncWorkspace(c.Request.Context(), workspaceID, layerID, notification.Bucket, notification.Name)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}

// parseObjectPath derives a workspaceID and layerID from a GCS object path.
//
//	"prod/network/default.tfstate"  → ("prod", "network")
//	"prod/default.tfstate"          → ("prod", "default")
//	"default.tfstate"               → ("default", "default")
func parseObjectPath(objectPath string) (workspaceID, layerID string) {
	// Strip the trailing filename to work with the directory segments only.
	lastSlash := strings.LastIndex(objectPath, "/")
	if lastSlash == -1 {
		// Flat file at bucket root: no workspace/layer context.
		name := strings.TrimSuffix(objectPath, ".tfstate")
		return name, "default"
	}

	dir := objectPath[:lastSlash] // e.g. "prod/network" or "prod"
	segments := strings.Split(dir, "/")

	switch len(segments) {
	case 1:
		// e.g. "prod/default.tfstate" → workspace=prod, layer=default
		return segments[0], "default"
	default:
		// e.g. "prod/network/default.tfstate" → workspace=prod, layer=network
		// Any deeper nesting uses the last two meaningful segments.
		return segments[len(segments)-2], segments[len(segments)-1]
	}
}
