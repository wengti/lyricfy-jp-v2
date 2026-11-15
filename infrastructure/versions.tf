terraform {
  required_version = ">= 1.10"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.9"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.14"
    }
    # The community provider that Neon's own docs use
    neon = {
      source  = "kislerdm/neon"
      version = "~> 0.18"
    }
  }

  # The bucket comes from bootstrap/: terraform init -backend-config="bucket=<project-id>-tfstate"
  backend "gcs" {
    prefix = "infrastructure"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# The Neon API key lives in Secret Manager (created by bootstrap/, filled in by hand), so the
# pipeline needs no key of its own: plan and apply read it as their Google identities. Read
# ephemerally, so it never lands in the state or the saved plan.
ephemeral "google_secret_manager_secret_version" "neon_api_key" {
  secret = "neon-api-key"
}

provider "neon" {
  api_key = ephemeral.google_secret_manager_secret_version.neon_api_key.secret_data
}

# The Billing Budgets API needs a quota project named on each request (budget.tf)
provider "google" {
  alias                 = "billing"
  project               = var.project_id
  billing_project       = var.project_id
  user_project_override = true
}
