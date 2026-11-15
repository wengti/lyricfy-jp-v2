terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.9"
    }
    neon = {
      source  = "kislerdm/neon"
      version = ">= 0.18"
    }
  }
}
