output "url" {
  description = "The public site."
  value       = google_cloud_run_v2_service.frontend.uri
}
