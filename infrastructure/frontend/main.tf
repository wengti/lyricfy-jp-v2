# The Next.js app on Cloud Run. Public, and the only caller allowed to reach the backend.
# Its service account and secret access come from ../identities.tf.

# The backend's only invoker
resource "google_cloud_run_v2_service_iam_member" "backend_invoker" {
  name     = var.backend_service_name
  location = var.backend_location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.service_account}"
}

resource "google_cloud_run_v2_service" "frontend" {
  name     = "lyricfy-frontend"
  location = var.region

  # Stateless and recreated by apply, so nothing to protect
  deletion_protection = false

  ingress = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = var.service_account

    # Scale to zero when idle; one instance is plenty for one user and caps the bill
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }

    containers {
      image = var.image

      # Matches the image's default PORT; Cloud Run also sets PORT to this
      ports {
        container_port = 3000
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle = true # CPU (and billing) only while handling a request
      }

      # Read by the Next.js server at request time (frontend/lib/api/client.ts)
      env {
        name  = "API_URL"
        value = var.backend_url
      }

      env {
        name = "FRONTEND_API_KEY"
        value_source {
          secret_key_ref {
            secret  = var.api_key_secret_id
            version = "latest"
          }
        }
      }
    }
  }

  lifecycle {
    # Redeploying from the Cloud Run console stamps these; they don't change the service
    ignore_changes = [client, client_version]
  }
}

# Public site: anyone may load the frontend (login still protects the app's pages)
resource "google_cloud_run_v2_service_iam_member" "public" {
  name     = google_cloud_run_v2_service.frontend.name
  location = google_cloud_run_v2_service.frontend.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
