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
		WorkspaceID   string `json:"workspace_id"`
		LayerID       string `json:"layer_id"`
		TFWorkspaceID string `json:"tf_workspace_id"` // e.g. "prod"; defaults to "default"
		Bucket        string `json:"bucket"`
		Object        string `json:"object"`
	}

	if err := c.DecodeJSON(&body); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	if body.LayerID == "" {
		body.LayerID = "default"
	}
	if body.TFWorkspaceID == "" {
		body.TFWorkspaceID = "default"
	}

	org, err := h.service.SyncWorkspace(c.Request.Context(), body.WorkspaceID, body.LayerID, body.TFWorkspaceID, body.Bucket, body.Object)
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
// GCS object path convention: {workspaceID}/{layerID}/{tfWorkspaceID}.tfstate
//   - "lz/apps/prod.tfstate"    → workspace="lz",      layer="apps",    tfWorkspace="prod"
//   - "lz/network/default.tfstate" → workspace="lz",   layer="network", tfWorkspace="default"
//   - "default.tfstate"         → workspace="default",  layer="default", tfWorkspace="default"
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

	workspaceID, layerID, tfWorkspaceID := parseObjectPath(notification.Name)

	org, err := h.service.SyncWorkspace(c.Request.Context(), workspaceID, layerID, tfWorkspaceID, notification.Bucket, notification.Name)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}

// parseObjectPath derives workspaceID, layerID, and tfWorkspaceID from a GCS object path.
// The Terraform workspace name is encoded in the filename (without .tfstate extension).
//
//	"lz/apps/prod.tfstate"        → ("lz",      "apps",    "prod")
//	"lz/network/default.tfstate"  → ("lz",      "network", "default")
//	"lz/default.tfstate"          → ("lz",      "default", "default")
//	"default.tfstate"             → ("default",  "default", "default")
func parseObjectPath(objectPath string) (workspaceID, layerID, tfWorkspaceID string) {
	lastSlash := strings.LastIndex(objectPath, "/")
	filename := strings.TrimSuffix(objectPath[lastSlash+1:], ".tfstate")
	if lastSlash == -1 {
		// Flat file at bucket root: workspace name = filename, no layer context.
		return filename, "default", "default"
	}

	// The TF workspace name is the filename (e.g. "prod" from "prod.tfstate").
	tfWorkspaceID = filename

	dir := objectPath[:lastSlash] // e.g. "lz/apps" or "lz"
	segments := strings.Split(dir, "/")

	switch len(segments) {
	case 1:
		// e.g. "lz/prod.tfstate" → workspace=lz, layer=default, tfWorkspace=prod
		return segments[0], "default", tfWorkspaceID
	default:
		// e.g. "lz/apps/prod.tfstate" → workspace=lz, layer=apps, tfWorkspace=prod
		return segments[len(segments)-2], segments[len(segments)-1], tfWorkspaceID
	}
}
