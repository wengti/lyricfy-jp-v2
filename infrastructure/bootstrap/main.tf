# One-time setup, applied locally: only what the pipeline needs before its first run.
#   main.tf      APIs and the Terraform state bucket
#   ci.tf        the identities the pipeline signs in as, one per stage
#   registry.tf  where the pipeline pushes images (before Terraform runs)
#   secrets.tf   secret containers, filled by hand before the first deploy
#   config.tf    the backend's config parameter, filled by hand before the first deploy
# Everything else, including every permission the services get, is in infrastructure/ and
# applied by the pipeline.

locals {
  # The APIs used by this config and by the pipeline's first run. The pipeline can enable
  # further APIs itself when new resources need them.
  services = [
    "artifactregistry.googleapis.com",
    "billingbudgets.googleapis.com",
    "cloudbilling.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com", # pipeline: short-lived tokens for the CI service account
    "parametermanager.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "storage.googleapis.com",
    "sts.googleapis.com", # pipeline: Workload Identity Federation
  ]
}

resource "google_project_service" "this" {
  for_each = toset(local.services)
  service  = each.value

  # Keep APIs on if this config is destroyed: everything else depends on them
  disable_on_destroy = false
}

# State for infrastructure/. State can hold secrets, so the bucket is private, and versioned
# so a bad apply can be rolled back.
resource "google_storage_bucket" "tfstate" {
  name     = "${var.project_id}-tfstate"
  location = var.state_bucket_location

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  # Keep the 10 most recent old versions of each state file
  lifecycle_rule {
    condition {
      num_newer_versions = 10
      with_state         = "ARCHIVED"
    }
    action {
      type = "Delete"
    }
  }

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.this]
}
