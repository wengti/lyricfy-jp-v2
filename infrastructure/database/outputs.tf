output "url_secret_id" {
  description = "Secret holding the connection string: the backend's DATABASE_URL."
  value       = google_secret_manager_secret.database_url.secret_id
}

output "url_secret_version" {
  description = "Version of that secret to use. Pinned, so a new connection string redeploys the backend."
  value       = google_secret_manager_secret_version.database_url.version
}
