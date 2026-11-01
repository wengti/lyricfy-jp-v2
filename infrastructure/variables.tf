# The pipeline sets these through TF_VAR_* environment variables (.github/workflows/pipeline.yml)

variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "Region for both services. Must match bootstrap/'s region, where the registry and secrets are."
  type        = string
  default     = "asia-southeast1"
}

variable "billing_account" {
  description = "Billing account ID linked to the project, e.g. 012345-6789AB-CDEF01."
  type        = string
}

variable "budget_amount" {
  description = "Monthly budget in the billing account's currency. Alerts fire at 10%, 50% and 100% of it."
  type        = number
  default     = 1

  validation {
    condition     = var.budget_amount >= 1 && floor(var.budget_amount) == var.budget_amount
    error_message = "budget_amount must be a whole number of at least 1."
  }
}

variable "backend_image" {
  description = "Backend image to deploy: <registry>/backend:<commit SHA>, pushed by the pipeline."
  type        = string
}

variable "frontend_image" {
  description = "Frontend image to deploy: <registry>/frontend:<commit SHA>, pushed by the pipeline."
  type        = string
}
