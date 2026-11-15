# Everything the pipeline applies, in one state:
#   identities.tf  the services' runtime identities and what they may access
#   main.tf        the two services and the database
#   budget.tf      the budget alert
# bootstrap/ (applied once, locally) must exist first: it holds the state bucket, the pipeline's
# identities, the image registry, and the secrets and config that are filled in by hand.

# Created by bootstrap/, referred to by name
locals {
  backend_config_parameter_id = "backend-config"
  secrets = {
    frontend_api_key = "frontend-api-key"
    # Unused since sessions moved to the database, but backend-config may still refer to it,
    # and rendering fails on a reference it can't read. Remove once backend-config doesn't.
    session_secret     = "session-secret"
    user_password_hash = "user-password-hash"
  }
}

module "database" {
  source = "./database"

  region = var.region
}

module "backend" {
  source = "./backend"

  region           = var.region
  image            = var.backend_image
  service_account  = google_service_account.backend.email
  config_parameter = data.google_parameter_manager_parameter.backend_config.name

  database_url_secret         = module.database.url_secret_id
  database_url_secret_version = module.database.url_secret_version

  depends_on = [time_sleep.iam_propagation]
}

module "frontend" {
  source = "./frontend"

  region            = var.region
  image             = var.frontend_image
  service_account   = google_service_account.frontend.email
  api_key_secret_id = local.secrets.frontend_api_key

  backend_url          = module.backend.url
  backend_service_name = module.backend.service_name
  backend_location     = module.backend.location

  depends_on = [time_sleep.iam_propagation]
}
