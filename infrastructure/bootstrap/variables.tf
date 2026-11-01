variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "Region for the image registry and secrets. Must match the pipeline's GCP_REGION, where the services run. asia-southeast1 (Singapore) is the closest to Malaysia, which has no region yet."
  type        = string
  default     = "asia-southeast1"
}

variable "state_bucket_location" {
  description = "Region for the Terraform state bucket. Cloud Storage's free tier only covers us-central1, us-east1 and us-west1."
  type        = string
  default     = "us-central1"
}

variable "billing_account" {
  description = "Billing account ID linked to the project, e.g. 012345-6789AB-CDEF01 (gcloud billing accounts list). The pipeline manages a budget on it."
  type        = string
}

variable "github_repository" {
  description = "GitHub repository (owner/name) whose pipeline on main may deploy."
  type        = string
  default     = "wengti/lyricfy-jp-v2"
}

variable "github_subject_prefix" {
  description = "Start of the subject claim in the repository's GitHub OIDC tokens. This repository uses GitHub's immutable format, repo:OWNER@OWNER_ID/NAME@REPO_ID, so a deleted and recreated repository with the same name can't deploy. Check it with: curl https://api.github.com/repos/OWNER/NAME/actions/oidc/customization/sub (sub_claim_prefix)."
  type        = string
  default     = "repo:wengti@45443501/lyricfy-jp-v2@1433809522"
}
