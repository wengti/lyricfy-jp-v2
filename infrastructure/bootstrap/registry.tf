# One Docker repository for both images: <region>-docker.pkg.dev/<project>/lyricfy/{backend,frontend}.
# Here rather than in infrastructure/ because the pipeline pushes images before Terraform runs.
resource "google_artifact_registry_repository" "images" {
  repository_id = "lyricfy"
  location      = var.region
  format        = "DOCKER"
  description   = "Lyricfy JP images, pushed by the pipeline and deployed to Cloud Run."

  # The free tier covers 0.5 GB of storage, so keep only the 2 newest versions of each image.
  # KEEP wins over DELETE; the policies run about once a day.
  cleanup_policy_dry_run = false

  cleanup_policies {
    id     = "keep-newest"
    action = "KEEP"
    most_recent_versions {
      keep_count = 2
    }
  }

  cleanup_policies {
    id     = "delete-older"
    action = "DELETE"
    condition {
      tag_state = "ANY"
    }
  }

  depends_on = [google_project_service.this]
}
