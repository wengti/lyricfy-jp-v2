output "service_name" {
  value = google_cloud_run_v2_service.backend.name
}

output "location" {
  value = google_cloud_run_v2_service.backend.location
}

output "url" {
  description = "The frontend's API_URL, and the audience of its identity tokens."
  value       = google_cloud_run_v2_service.backend.uri
}
