# Containers only: values are added with gcloud so they never pass through Terraform state.
# Here rather than in infrastructure/ because the services can't start until these have values,
# so they must exist (and be filled) before the pipeline's first deploy. Who may read them is
# granted by the pipeline (infrastructure/identities.tf).

locals {
  secrets = {
    # frontend.api_key in the backend config, and the frontend's FRONTEND_API_KEY: one secret,
    # so the two can't drift apart
    frontend_api_key = "frontend-api-key"
    # session.secret in the backend config
    session_secret = "session-secret"
    # user.password_hash in the backend config. A bcrypt hash, but kept secret anyway: a leaked
    # hash can be brute-forced offline.
    user_password_hash = "user-password-hash"
  }
}

resource "google_secret_manager_secret" "this" {
  for_each  = local.secrets
  secret_id = each.value

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  depends_on = [google_project_service.this]
}
