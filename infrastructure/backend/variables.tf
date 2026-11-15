variable "region" {
  description = "Region to run the service in."
  type        = string
}

variable "image" {
  description = "Image to deploy."
  type        = string
}

variable "service_account" {
  description = "Email of the service account the service runs as (created in ../identities.tf)."
  type        = string
}

variable "config_parameter" {
  description = "Parameter Manager parameter holding the backend's config: projects/PROJECT/locations/global/parameters/NAME."
  type        = string
}

variable "database_url_secret" {
  description = "Secret holding the database connection string, read into DATABASE_URL (created in ../database)."
  type        = string
}

variable "database_url_secret_version" {
  description = "Version of the connection string secret to read."
  type        = string
}
