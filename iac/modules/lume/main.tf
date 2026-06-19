terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }
}
# ── Required GCP APIs ─────────────────────────────────────────────────────────

resource "google_project_service" "run" {
  project            = var.project_id
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "firestore" {
  project            = var.project_id
  service            = "firestore.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "iam" {
  project            = var.project_id
  service            = "iam.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "iap" {
  project            = var.project_id
  service            = "iap.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "compute" {
  project            = var.project_id
  service            = "compute.googleapis.com"
  disable_on_destroy = false
}

# ── Project data (needed for IAP service agent) ───────────────────────────────

data "google_project" "project" {
  project_id = var.project_id
}

# ── Artifact Registry ─────────────────────────────────────────────────────────

data "google_artifact_registry_repository" "lume" {
  project       = local.artifact_registry_project
  location      = local.artifact_registry_location
  repository_id = local.artifact_registry_repo
}

# Resolve the mutable image tags to their current immutable digests. Referencing
# self_link (which embeds the @sha256:... digest) from the Cloud Run services
# means a retag of the same tag changes the resolved digest on the next plan and
# forces a redeploy.
data "google_artifact_registry_docker_image" "backend" {
  project       = local.artifact_registry_project
  location      = local.artifact_registry_location
  repository_id = local.artifact_registry_repo
  image_name    = local.backend_image_name
}

data "google_artifact_registry_docker_image" "frontend" {
  project       = local.artifact_registry_project
  location      = local.artifact_registry_location
  repository_id = local.artifact_registry_repo
  image_name    = local.frontend_image_name
}

resource "google_artifact_registry_repository_iam_member" "main" {
  member     = "serviceAccount:service-${data.google_project.project.number}@serverless-robot-prod.iam.gserviceaccount.com"
  project    = data.google_artifact_registry_repository.lume.project
  location   = data.google_artifact_registry_repository.lume.location
  repository = data.google_artifact_registry_repository.lume.name
  role       = "roles/artifactregistry.reader"
}

# ── Firestore (Native mode) ───────────────────────────────────────────────────
# Uses the default database so the Go backend (firestore.NewClient) works
# without any code changes. Each environment lives in its own GCP project.

resource "google_firestore_database" "lume" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"

  # Prevent accidental deletion of persistent data.
  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.firestore]
}

# ── Backend service account ───────────────────────────────────────────────────

resource "google_service_account" "backend" {
  project      = var.project_id
  account_id   = "lume-backend-${var.environment}"
  display_name = "Lume Backend SA (${var.environment})"
  description  = "Identity for the lume-backend Cloud Run service."

  depends_on = [google_project_service.iam]
}

# Firestore read/write
resource "google_project_iam_member" "backend_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.backend.email}"
}

# Artifact Registry read (image pull at Cloud Run start-up)
resource "google_project_iam_member" "backend_registry_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.backend.email}"
}

# ── Frontend service account ──────────────────────────────────────────────────

resource "google_service_account" "frontend" {
  project      = var.project_id
  account_id   = "lume-frontend-${var.environment}"
  display_name = "Lume Frontend SA (${var.environment})"
  description  = "Identity for the lume-frontend Cloud Run service."

  depends_on = [google_project_service.iam]
}

# Artifact Registry read (image pull at Cloud Run start-up)
resource "google_project_iam_member" "frontend_registry_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.frontend.email}"
}

# ── Backend Cloud Run service ─────────────────────────────────────────────────

resource "google_cloud_run_v2_service" "backend" {
  project             = var.project_id
  name                = local.backend_service_name
  location            = var.region
  deletion_protection = false

  # Accept traffic only from within the project / VPC; Cloud Run URL is never
  # exposed to the public internet directly.
  ingress = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.backend.email

    scaling {
      min_instance_count = var.backend_min_instances
      max_instance_count = var.backend_max_instances
    }

    containers {
      image = data.google_artifact_registry_docker_image.backend.self_link

      ports {
        container_port = 3000
      }

      resources {
        limits = {
          cpu    = var.backend_cpu
          memory = var.backend_memory
        }
      }

      # GCP project ID used by the Firestore and GCS clients.
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }

      # GCS bucket name passed to the state-file downloader.
      # Bucket IAM is managed outside this module.
      env {
        name  = "GCS_BUCKET"
        value = var.bucket_name
      }
    }
  }

  depends_on = [google_project_service.run, google_artifact_registry_repository_iam_member.main]
}

# ── VPC for frontend → backend Direct VPC egress ──────────────────────────────
# The backend uses INGRESS_TRAFFIC_INTERNAL_ONLY, so the frontend's outbound
# call must travel over a VPC (rather than the public internet) to be treated as
# internal traffic. Direct VPC egress attaches the frontend instances to this
# subnet; egress = "ALL_TRAFFIC" routes the backend *.run.app request through it.

resource "google_compute_network" "main" {
  project                 = var.project_id
  name                    = local.network_name
  auto_create_subnetworks = false

  depends_on = [google_project_service.compute]
}

resource "google_compute_subnetwork" "frontend_egress" {
  project       = var.project_id
  name          = local.frontend_egress_subnet_name
  region        = var.region
  network       = google_compute_network.main.id
  ip_cidr_range = var.frontend_egress_subnet_cidr

  # Required so the frontend (no external IP on the VPC-egress path) can reach
  # Google services — including the backend's *.run.app front end — over
  # Google's internal network. Without this the TCP connection is dropped and
  # the proxy fails with "fetch failed".
  private_ip_google_access = true
}

# ── Frontend Cloud Run service ────────────────────────────────────────────────
# NEXT_PUBLIC_API_URL is set to the backend service URI returned by Cloud Run.
# IAP is enabled natively: unauthenticated users are redirected to the Google
# sign-in flow before any request reaches the container.

resource "google_cloud_run_v2_service" "frontend" {
  provider            = google-beta
  project             = var.project_id
  name                = local.frontend_service_name
  location            = var.region
  deletion_protection = false
  launch_stage        = "BETA"

  # Accept all incoming traffic — IAP is the authentication boundary.
  ingress     = "INGRESS_TRAFFIC_ALL"
  iap_enabled = true

  template {
    service_account = google_service_account.frontend.email

    scaling {
      min_instance_count = var.frontend_min_instances
      max_instance_count = var.frontend_max_instances
    }

    # Direct VPC egress: route all outbound traffic through the VPC so requests
    # to the internal-only backend are treated as internal.
    vpc_access {
      network_interfaces {
        network    = google_compute_network.main.id
        subnetwork = google_compute_subnetwork.frontend_egress.id
      }
      egress = "ALL_TRAFFIC"
    }

    containers {
      image = data.google_artifact_registry_docker_image.frontend.self_link

      ports {
        container_port = 3000
      }

      resources {
        limits = {
          cpu    = var.frontend_cpu
          memory = var.frontend_memory
        }
      }

      env {
        name  = "API_URL"
        value = google_cloud_run_v2_service.backend.uri
      }
    }
  }

  depends_on = [
    google_project_service.run,
    google_project_service.iap,
    google_compute_subnetwork.frontend_egress,
    google_artifact_registry_repository_iam_member.main
  ]
}

# ── IAP → Frontend: allow the IAP service agent to invoke the Cloud Run service ─

resource "google_cloud_run_v2_service_iam_member" "frontend_iap_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.frontend.name
  role     = "roles/run.invoker"
  # IAP service agent for this project
  member = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-iap.iam.gserviceaccount.com"

  depends_on = [google_project_service.iap]
}

# ── Frontend SA → Backend: only the frontend may call the backend ─────────────

resource "google_cloud_run_v2_service_iam_member" "backend_frontend_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.backend.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.frontend.email}"
}
