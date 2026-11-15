# The Postgres database on Neon, and its connection string in Secret Manager for the backend.
# The backend applies its own schema migrations when it starts (backend/internal/database).
# Who may read the connection string is granted in ../identities.tf.

resource "neon_project" "this" {
  name = "lyricfy"

  # Singapore, next to the Cloud Run services in asia-southeast1
  region_id  = "aws-ap-southeast-1"
  pg_version = 17 # match backend/compose.yaml, the local database

  # Point-in-time restore window. The Free plan allows at most 6 hours, and the provider's
  # default (1 day) would fail there.
  history_retention_seconds = 21600

  branch {
    name          = "main"
    database_name = "lyricfy"
    role_name     = "lyricfy"
  }

  # Holds the app's data: never destroy it by accident. To really delete it, remove this first.
  lifecycle {
    prevent_destroy = true
  }
}

resource "google_secret_manager_secret" "database_url" {
  secret_id = "database-url"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }
}

# The direct (unpooled) connection string: the backend's migrations take a session-level lock,
# which Neon's pooler (PgBouncer in transaction mode) doesn't support. One Cloud Run instance
# with a small pool is well within Neon's connection limit.
resource "google_secret_manager_secret_version" "database_url" {
  secret      = google_secret_manager_secret.database_url.id
  secret_data = neon_project.this.connection_uri
}
