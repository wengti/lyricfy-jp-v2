# The pipeline signs in with Workload Identity Federation: a job exchanges its GitHub OIDC token
# for a short-lived token of one of the service accounts below, so no keys are stored.
#
# One service account per stage, each with only what that stage needs:
#   github-build  push images                      any job on main
#   github-plan   read everything, change nothing  any job on main
#   github-apply  Owner: change anything           only jobs in the "production" environment,
#                                                  which GitHub runs only after a reviewer approves
# So code that reaches main can push an image or read the project, but nothing changes until
# someone approves the plan.

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github"
  display_name              = "GitHub Actions"

  depends_on = [google_project_service.this]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-actions"
  display_name                       = "GitHub Actions OIDC"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  # GitHub's subject claim names the repository and either the branch or the environment:
  #   repo:OWNER/NAME:ref:refs/heads/main      a job on main
  #   repo:OWNER/NAME:environment:production   a job in the production environment
  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
    "attribute.ref"        = "assertion.ref"
  }

  # Only this repository's pipeline running on main gets in. Pull requests test and build
  # without cloud access, so a PR from a branch or fork can never deploy.
  attribute_condition = "assertion.repository == '${var.github_repository}' && assertion.ref == 'refs/heads/main'"
}

locals {
  github_principal = "principal://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/subject"
  # Which GitHub jobs may act as each service account (see the subject formats above)
  github_subjects = {
    main       = "${local.github_principal}/repo:${var.github_repository}:ref:refs/heads/main"
    production = "${local.github_principal}/repo:${var.github_repository}:environment:production"
  }
}

# ---------------------------------------------------------------------------
# github-build: push images
# ---------------------------------------------------------------------------

resource "google_service_account" "build" {
  account_id   = "github-build"
  display_name = "GitHub Actions: build images"

  depends_on = [google_project_service.this]
}

resource "google_service_account_iam_member" "build_federation" {
  service_account_id = google_service_account.build.name
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_subjects.main
}

resource "google_artifact_registry_repository_iam_member" "build_push" {
  location   = google_artifact_registry_repository.images.location
  repository = google_artifact_registry_repository.images.name
  role       = "roles/artifactregistry.writer"
  member     = google_service_account.build.member
}

# ---------------------------------------------------------------------------
# github-plan: read everything Terraform manages, change nothing
# ---------------------------------------------------------------------------

resource "google_service_account" "plan" {
  account_id   = "github-plan"
  display_name = "GitHub Actions: terraform plan (read-only)"

  depends_on = [google_project_service.this]
}

resource "google_service_account_iam_member" "plan_federation" {
  service_account_id = google_service_account.plan.name
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_subjects.main
}

locals {
  plan_project_roles = [
    "roles/viewer",                            # read resources
    "roles/iam.securityReviewer",              # read IAM policies on them
    "roles/serviceusage.serviceUsageConsumer", # bill API calls to this project (Budget API)
  ]
}

resource "google_project_iam_member" "plan" {
  for_each = toset(local.plan_project_roles)
  project  = var.project_id
  role     = each.value
  member   = google_service_account.plan.member
}

# Plan takes a lock in the state bucket, so it writes there, but only there
resource "google_storage_bucket_iam_member" "plan_state" {
  bucket = google_storage_bucket.tfstate.name
  role   = "roles/storage.objectAdmin"
  member = google_service_account.plan.member
}

# Read the budget alert (infrastructure/budget.tf)
resource "google_billing_account_iam_member" "plan_budgets" {
  billing_account_id = var.billing_account
  role               = "roles/billing.viewer"
  member             = google_service_account.plan.member

  depends_on = [google_project_service.this]
}

# ---------------------------------------------------------------------------
# github-apply: change anything, but only after approval
# ---------------------------------------------------------------------------

resource "google_service_account" "apply" {
  account_id   = "github-apply"
  display_name = "GitHub Actions: terraform apply (after approval)"

  depends_on = [google_project_service.this]
}

# Only jobs in the production environment, which waits for a required reviewer
resource "google_service_account_iam_member" "apply_federation" {
  service_account_id = google_service_account.apply.name
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_subjects.production
}

# Owner, so new resources and their permissions (including project-wide ones) never need a
# change here. The approval step is what guards it: read the plan before approving.
resource "google_project_iam_member" "apply_owner" {
  project = var.project_id
  role    = "roles/owner"
  member  = google_service_account.apply.member
}

# Manage the budget alert. Billing accounts sit outside the project, so Owner doesn't cover it.
resource "google_billing_account_iam_member" "apply_budgets" {
  billing_account_id = var.billing_account
  role               = "roles/billing.costsManager"
  member             = google_service_account.apply.member

  depends_on = [google_project_service.this]
}
