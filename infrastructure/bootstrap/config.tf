# The backend's config.yml, edited in the console (Parameter Manager → backend-config → New
# version). The backend loads the newest version when an instance starts, so a new version
# takes effect on the next deploy. Secret values aren't stored here: the YAML refers to them
# with __REF__(...), and Parameter Manager fills them in when the backend reads it.
# See backend/config.parameter.example.yml.
#
# Terraform creates the parameter only, never its versions, so console edits don't fight it.
# Its permission to read the secrets is granted by the pipeline (infrastructure/identities.tf).
# Global, because Parameter Manager has no regional option in asia-southeast1.
resource "google_parameter_manager_parameter" "backend_config" {
  parameter_id = "backend-config"
  format       = "YAML" # rejects versions that aren't valid YAML

  depends_on = [google_project_service.this]
}
