# The Go API on Cloud Run. Private: only the frontend's service account may call it
# (granted in ../frontend/main.tf). Its service account and config access come from ../identities.tf.

resource "google_cloud_run_v2_service" "backend" {
  name     = "lyricfy-backend"
  location = var.region

  # Stateless and recreated by apply, so nothing to protect
  deletion_protection = false

  # Reachable over the internet, but every request needs an identity token for an account with
  # roles/run.invoker, and there is no allUsers binding
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

      resources {
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
        cpu_idle = true # CPU (and billing) only while handling a request
      }

      # The backend loads the newest version of this parameter at startup, so a config edit in
      # the console takes effect on the next deploy (or cold start)
      env {
        name  = "CONFIG_PARAMETER"
        value = var.config_parameter
      }
    }
  }

  lifecycle {
    # Redeploying from the Cloud Run console (to pick up a config edit) stamps these; they don't
    # change the service
    ignore_changes = [client, client_version]
  }
}
