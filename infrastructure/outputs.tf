output "frontend_url" {
  description = "The public site."
  value       = module.frontend.url
}

output "backend_url" {
  description = "The private API (only the frontend may call it)."
  value       = module.backend.url
}
