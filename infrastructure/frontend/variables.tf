variable "region" {
  description = "Region to run the service in."
  type        = string
}

variable "image" {
  description = "Image to deploy."
  type        = string
}

variable "service_account" {
  description = "Email of the service account the service runs as (created by bootstrap/)."
  type        = string
}

variable "api_key_secret_id" {
  description = "Secret holding FRONTEND_API_KEY."
  type        = string
}

variable "backend_url" {
  description = "The backend's URL: the frontend's API_URL."
  type        = string
}

variable "backend_service_name" {
  description = "The backend's Cloud Run service, which the frontend is allowed to invoke."
  type        = string
}

variable "backend_location" {
  description = "The backend service's region."
  type        = string
}
