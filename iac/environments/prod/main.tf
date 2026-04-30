variable "project_id" {
  description = "GCP project ID for the prod environment."
  type        = string
}


variable "region" {
  description = "GCP region for all resources."
  type        = string
  default     = "europe-west1"
}

variable "bucket_name" {
  description = "GCS bucket containing Terraform state files to observe."
  type        = string
}

variable "artifact_registry_url" {
  description = <<-EOT
        Base URL of the Artifact Registry repository used to build default image references, e.g.
        'europe-docker.pkg.dev/my-project/lume'.
        When set, the default backend and frontend images are derived from this URL
        ('<artifact_registry_url>/backend:latest' and '<artifact_registry_url>/frontend:latest').
        Ignored when backend_image or frontend_image are set explicitly.
        When null, defaults to '<region>-docker.pkg.dev/<project_id>/lume'.
    EOT
  type        = string
  default     = null
}

# ── Module call ───────────────────────────────────────────────────────────────

module "lume" {
  source = "../../modules/lume"

  project_id  = var.project_id
  region      = var.region
  environment = "prod"
  bucket_name = var.bucket_name

  artifact_registry_url = var.artifact_registry_url
  # Prod: higher baseline capacity, keep at least 1 instance warm to avoid
  # cold-start latency.
  backend_cpu           = "2"
  backend_memory        = "1Gi"
  backend_min_instances = 0
  backend_max_instances = 20

  frontend_cpu           = "1"
  frontend_memory        = "512Mi"
  frontend_min_instances = 0
  frontend_max_instances = 20
}

# ── Outputs ───────────────────────────────────────────────────────────────────

output "backend_url" {
  description = "Backend Cloud Run URL."
  value       = module.lume.backend_url
}


output "frontend_url" {
  description = "Frontend Cloud Run URL."
  value       = module.lume.frontend_url
}

output "artifact_registry_url" {
  description = "Artifact Registry base URL for pushing images."
  value       = module.lume.artifact_registry_url
}

output "backend_service_account_email" {
  description = "Backend service account email."
  value       = module.lume.backend_service_account_email
}

output "frontend_service_account_email" {
  description = "Frontend service account email."
  value       = module.lume.frontend_service_account_email
}

