terraform {
  required_version = ">= 1.10"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.9"
    }
  }

  # No backend block: this config creates the state bucket, so its own state stays local
  # (terraform.tfstate, gitignored). See "Infrastructure" in the root README.md.
}

provider "google" {
  project = var.project_id
  region  = var.region
}
