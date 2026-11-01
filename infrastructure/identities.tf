# The identities the two Cloud Run services run as, and everything they may access.
# To give a service access to something new, grant it here.

resource "google_service_account" "backend" {
  account_id   = "lyricfy-backend"
  display_name = "Lyricfy backend (Cloud Run)"
}

resource "google_service_account" "frontend" {
  account_id   = "lyricfy-frontend"
  display_name = "Lyricfy frontend (Cloud Run)"
}

# ---------------------------------------------------------------------------
# Backend
# ---------------------------------------------------------------------------

# Render its config. Parameter Manager only grants project-wide, but backend-config is the
# only parameter.
resource "google_project_iam_member" "backend_config" {
  project = var.project_id
  role    = "roles/parametermanager.parameterAccessor"
  member  = google_service_account.backend.member
}

# The config parameter (created by bootstrap/) resolves its __REF__(...) secret references as
# its own identity, so it needs to read the secrets
data "google_parameter_manager_parameter" "backend_config" {
  parameter_id = local.backend_config_parameter_id
}

resource "google_secret_manager_secret_iam_member" "backend_config" {
  for_each  = local.secrets
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = data.google_parameter_manager_parameter.backend_config.policy_member[0].iam_policy_uid_principal
}

# ---------------------------------------------------------------------------
# Frontend
# ---------------------------------------------------------------------------

# Read its API key (env var FRONTEND_API_KEY)
resource "google_secret_manager_secret_iam_member" "frontend_api_key" {
  secret_id = local.secrets.frontend_api_key
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.frontend.member
}

# ---------------------------------------------------------------------------

# New permissions can take a minute to reach every Google service. The services read their
# config and secrets as soon as they start, so give grants time to spread before (re)deploying
# them. Waits only when the grants change.
resource "time_sleep" "iam_propagation" {
  create_duration = "60s"

  triggers = {
    backend_config   = google_project_iam_member.backend_config.id
    config_secrets   = join(",", [for m in google_secret_manager_secret_iam_member.backend_config : m.id])
    frontend_api_key = google_secret_manager_secret_iam_member.frontend_api_key.id
    backend_account  = google_service_account.backend.email
    frontend_account = google_service_account.frontend.email
  }
}
