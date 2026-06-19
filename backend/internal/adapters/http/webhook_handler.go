package http

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/lume/backend/internal/core/ports"
	"github.com/mrshabel/mach"
)

type WebhookHandler struct {
	service ports.TofuService
	logger  *slog.Logger
}

func NewWebhookHandler(service ports.TofuService, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{service: service, logger: logger.With("component", "webhook_handler")}
}

func (h *WebhookHandler) Register(app *mach.App) {
	app.POST("/api/v1/webhooks/git", h.HandleWebhook)
	app.POST("/api/v1/webhooks/gcs", h.HandleGCSNotification)
}

func (h *WebhookHandler) HandleWebhook(c *mach.Context) {
	var body struct {
		LayerID       string `json:"layer_id"`
		TFWorkspaceID string `json:"tf_workspace_id"` // e.g. "prod"; defaults to "default"
		Bucket        string `json:"bucket"`
		Object        string `json:"object"`
	}

	ctx := c.Request.Context()
	if err := c.DecodeJSON(&body); err != nil {
		h.logger.WarnContext(ctx, "invalid webhook body", "route", "POST /api/v1/webhooks/git", "status", 400, "error", err)
		c.JSON(400, map[string]string{"error": "invalid request body"})
		return
	}

	if body.LayerID == "" {
		body.LayerID = "default"
	}
	if body.TFWorkspaceID == "" {
		body.TFWorkspaceID = "default"
	}

	org, err := h.service.SyncWorkspace(ctx, body.LayerID, body.TFWorkspaceID, body.Bucket, body.Object)
	if err != nil {
		h.logger.ErrorContext(ctx, "webhook sync failed", "route", "POST /api/v1/webhooks/git", "status", 500,
			"layer_id", body.LayerID, "tf_workspace", body.TFWorkspaceID, "bucket", body.Bucket, "object", body.Object, "error", err)
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
	Name      string `json:"name"`      // full object path, e.g. apps/prod.tfstate
	EventType string `json:"eventType"` // e.g. OBJECT_FINALIZE
}

// HandleGCSNotification handles Pub/Sub push messages sent by GCS object change notifications.
// Only OBJECT_FINALIZE events for .tfstate files trigger a hierarchy sync; all other events
// are acknowledged with 200 OK and ignored.
//
// GCS object path convention: {layerID}/{tfWorkspaceID}.tfstate
//   - "apps/prod.tfstate"    → layer="apps",    tfWorkspace="prod"
//   - "network/default.tfstate" → layer="network", tfWorkspace="default"
//   - "prod.tfstate"         → layer="default",  tfWorkspace="prod"
func (h *WebhookHandler) HandleGCSNotification(c *mach.Context) {
	ctx := c.Request.Context()
	log := h.logger.With("route", "POST /api/v1/webhooks/gcs")

	var envelope pubSubMessage
	if err := c.DecodeJSON(&envelope); err != nil {
		log.WarnContext(ctx, "invalid pubsub envelope", "status", 400, "error", err)
		c.JSON(400, map[string]string{"error": "invalid pubsub envelope"})
		return
	}

	rawData, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		log.WarnContext(ctx, "invalid base64 in message.data", "status", 400, "error", err)
		c.JSON(400, map[string]string{"error": "invalid base64 in message.data"})
		return
	}

	var notification gcsNotification
	if err := json.Unmarshal(rawData, &notification); err != nil {
		log.WarnContext(ctx, "invalid gcs notification payload", "status", 400, "error", err)
		c.JSON(400, map[string]string{"error": "invalid gcs notification payload"})
		return
	}

	// Only process object creation/upload events for state files.
	if notification.EventType != "OBJECT_FINALIZE" || !strings.HasSuffix(notification.Name, ".tfstate") {
		log.DebugContext(ctx, "ignoring gcs notification", "event_type", notification.EventType, "object", notification.Name)
		c.JSON(200, map[string]string{"status": "ignored"})
		return
	}

	layerID, tfWorkspaceID := parseObjectPath(notification.Name)
	log.InfoContext(ctx, "processing gcs notification",
		"event_type", notification.EventType, "bucket", notification.Bucket, "object", notification.Name,
		"layer_id", layerID, "tf_workspace", tfWorkspaceID)

	org, err := h.service.SyncWorkspace(ctx, layerID, tfWorkspaceID, notification.Bucket, notification.Name)
	if err != nil {
		log.ErrorContext(ctx, "gcs notification sync failed", "status", 500,
			"bucket", notification.Bucket, "object", notification.Name, "error", err)
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, org)
}

// parseObjectPath derives layerID and tfWorkspaceID from a GCS object path.
// The Terraform workspace name is encoded in the filename (without .tfstate extension).
//
//	"apps/prod.tfstate"           → ("apps",    "prod")
//	"network/default.tfstate"     → ("network", "default")
//	"prod.tfstate"                → ("default", "prod")
func parseObjectPath(objectPath string) (layerID, tfWorkspaceID string) {
	lastSlash := strings.LastIndex(objectPath, "/")
	filename := strings.TrimSuffix(objectPath[lastSlash+1:], ".tfstate")
	if lastSlash == -1 {
		// Flat file at bucket root: tfWorkspace = filename, layer = default.
		return "default", filename
	}

	// The TF workspace name is the filename (e.g. "prod" from "prod.tfstate").
	tfWorkspaceID = filename
	// The last path segment before the filename is the layer ID.
	dir := objectPath[:lastSlash]
	segments := strings.Split(dir, "/")
	layerID = segments[len(segments)-1]
	return layerID, tfWorkspaceID
}
