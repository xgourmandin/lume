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

  # Parse project, location, and repository ID from the registry base URL.
  # URL format: <location>-docker.pkg.dev/<project_id>/<repository_id>
  _registry_parts            = split("/", local.registry_base)
  artifact_registry_location = split("-docker.pkg.dev", local._registry_parts[0])[0]
  artifact_registry_project  = local._registry_parts[1]
  artifact_registry_repo     = local._registry_parts[2]

  # ── Image resolution ───────────────────────────────────────────────────────
  # Use the caller-supplied image if provided; otherwise fall back to the
  # resolved registry base (tag: latest).
  backend_image_url  = var.backend_image != null ? var.backend_image : "${local.registry_base}/backend:latest"
  frontend_image_url = var.frontend_image != null ? var.frontend_image : "${local.registry_base}/frontend:latest"

  # Repository-relative image name (e.g. "backend:latest"), i.e. everything
  # after the "<location>-docker.pkg.dev/<project>/<repo>/" prefix. Consumed by
  # the google_artifact_registry_docker_image data source to resolve the tag to
  # an immutable digest so retags trigger a redeploy.
  backend_image_name  = join("/", slice(split("/", local.backend_image_url), 3, length(split("/", local.backend_image_url))))
  frontend_image_name = join("/", slice(split("/", local.frontend_image_url), 3, length(split("/", local.frontend_image_url))))
}

