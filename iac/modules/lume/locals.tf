locals {
  # ── Service names ───────────────────────────────────────────────────────────
  # Each environment lives in a distinct GCP project, so service names are the
  # same across environments. The project boundary provides isolation.
  backend_service_name  = "lume-backend"
  frontend_service_name = "lume-frontend"

  # ── Artifact Registry ──────────────────────────────────────────────────────
  # Use caller-supplied registry URL if provided; otherwise build the default
  # from region + project_id (the repository created by this module).
  registry_base = var.artifact_registry_url != null ? var.artifact_registry_url : "${var.region}-docker.pkg.dev/${var.project_id}/lume"

  # ── Image resolution ───────────────────────────────────────────────────────
  # Use the caller-supplied image if provided; otherwise fall back to the
  # resolved registry base (tag: latest).
  backend_image_url  = var.backend_image != null ? var.backend_image : "${local.registry_base}/backend:latest"
  frontend_image_url = var.frontend_image != null ? var.frontend_image : "${local.registry_base}/frontend:latest"
}

