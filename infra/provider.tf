terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # State lives in GCS: Cloud Shell's home directory is ephemeral, and losing
  # state after the resources exist means re-importing everything.
  backend "gcs" {
    bucket = "img-svc-train-tfstate"
    prefix = "foundation"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
