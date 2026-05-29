output "backend_url" {
  description = "Cloud Run URI for the backend service (available after apply)."
  value       = google_cloud_run_v2_service.backend.uri
}

output "frontend_url" {
  description = "Cloud Run URI for the frontend service (available after apply)."
  value       = google_cloud_run_v2_service.frontend.uri
}

output "backend_image_url" {
  description = "Full Docker image URL configured for the backend Cloud Run service."
  value       = local.backend_image_url
}

output "frontend_image_url" {
  description = "Full Docker image URL configured for the frontend Cloud Run service."
  value       = local.frontend_image_url
}

output "firestore_database" {
  description = "Name of the Firestore database provisioned for this environment."
  value       = google_firestore_database.lume.name
}

output "backend_service_account_email" {
  description = "Email of the service account assigned to the backend Cloud Run service."
  value       = google_service_account.backend.email
}

output "frontend_service_account_email" {
  description = "Email of the service account assigned to the frontend Cloud Run service."
  value       = google_service_account.frontend.email
}
