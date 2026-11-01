# Values for the GitHub repository variables the pipeline reads (see the root README.md). The
# pipeline derives its service account emails from GCP_PROJECT_ID.

output "region" {
  description = "GCP_REGION"
  value       = var.region
}

output "workload_identity_provider" {
  description = "GCP_WORKLOAD_IDENTITY_PROVIDER"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "state_bucket" {
  description = "Holds infrastructure/'s state. The pipeline derives it from the project ID."
  value       = google_storage_bucket.tfstate.name
}

output "backend_config_parameter" {
  description = "Edit the backend config here (console: Secret Manager → Parameter Manager)."
  value       = google_parameter_manager_parameter.backend_config.name
}

output "image_repository" {
  description = "Where the pipeline pushes images. It derives it from GCP_PROJECT_ID and GCP_REGION."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}"
}
