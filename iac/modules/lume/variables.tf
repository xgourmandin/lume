variable "project_id" {
  description = "GCP project ID where resources will be deployed."
  type        = string
}


variable "region" {
  description = "GCP region for Cloud Run services and Artifact Registry (e.g. 'europe-west1')."
  type        = string
}

variable "environment" {
  description = "Deployment environment. Must be 'dev' or 'prod'."
  type        = string
  validation {
    condition     = contains(["dev", "prod"], var.environment)
    error_message = "environment must be 'dev' or 'prod'."
  }
}

variable "bucket_name" {
  description = "Name of the GCS bucket containing Terraform state files to observe. Bucket IAM permissions are managed outside this module."
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

variable "backend_image" {
  description = <<-EOT
    Full Docker image reference for the backend (lume Go API), e.g.
    'europe-west1-docker.pkg.dev/my-project/lume/backend:1.0.0'.
    When null, defaults to '<artifact_registry_url>/backend:latest'.
  EOT
  type        = string
  default     = null
}

variable "frontend_image" {
  description = <<-EOT
    Full Docker image reference for the frontend (Next.js), e.g.
    'europe-west1-docker.pkg.dev/my-project/lume/frontend:1.0.0'.
    When null, defaults to '<artifact_registry_url>/frontend:latest'.
  EOT
  type        = string
  default     = null
}

# ── Backend sizing ──────────────────────────────────────────────────────────

variable "backend_cpu" {
  description = "vCPU limit for the backend Cloud Run service (e.g. '1', '2')."
  type        = string
  default     = "1"
}

variable "backend_memory" {
  description = "Memory limit for the backend Cloud Run service (e.g. '512Mi', '1Gi')."
  type        = string
  default     = "512Mi"
}

variable "backend_min_instances" {
  description = "Minimum number of backend instances. 0 enables scale-to-zero."
  type        = number
  default     = 0
}

variable "backend_max_instances" {
  description = "Maximum number of backend instances."
  type        = number
  default     = 10
}

# ── Frontend sizing ─────────────────────────────────────────────────────────

variable "frontend_cpu" {
  description = "vCPU limit for the frontend Cloud Run service."
  type        = string
  default     = "1"
}

variable "frontend_memory" {
  description = "Memory limit for the frontend Cloud Run service."
  type        = string
  default     = "512Mi"
}

variable "frontend_min_instances" {
  description = "Minimum number of frontend instances. 0 enables scale-to-zero."
  type        = number
  default     = 0
}

variable "frontend_max_instances" {
  description = "Maximum number of frontend instances."
  type        = number
  default     = 10
}

