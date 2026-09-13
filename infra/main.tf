# Foundational infrastructure for go-image-service.
#
# Cloud Run services and the worker pool are deliberately absent: those are
# deployed by .github/workflows/deploy.yml on every push to main, with a new
# image tag each time. Managing them here too would mean every apply reverts
# the running image and every deploy shows up as drift.

locals {
  services = [
    "artifactregistry.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "sqladmin.googleapis.com",
    "storage.googleapis.com",
  ]
}

resource "google_project_service" "enabled" {
  for_each = toset(local.services)

  project = var.project_id
  service = each.value

  # Leaving an API enabled is safer than breaking whatever still uses it.
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "services" {
  project       = var.project_id
  location      = var.region
  repository_id = "services"
  format        = "DOCKER"
  description   = "Container images for the image service, pushed by CI."
}

locals {
  service_accounts = {
    imageservice    = "Stateless resize gRPC service"
    authservice     = "Registration, login, token verification"
    httpserver      = "Public HTTP API and demo frontend"
    worker          = "Temporal worker running the upload pipeline"
    github-deployer = "CI deployer, impersonated via Workload Identity Federation"
    gcs-hmac        = "Owns the HMAC keys used for S3-compatible presigned URLs"
  }
}

resource "google_service_account" "accounts" {
  for_each = local.service_accounts

  project      = var.project_id
  account_id   = "${each.key}-sa"
  display_name = each.value
}

locals {
  # Keyed "<sa>|<role>" so each binding is its own resource instance.
  project_roles = {
    "authservice|roles/cloudsql.client"          = { sa = "authservice", role = "roles/cloudsql.client" }
    "httpserver|roles/cloudsql.client"           = { sa = "httpserver", role = "roles/cloudsql.client" }
    "worker|roles/cloudsql.client"               = { sa = "worker", role = "roles/cloudsql.client" }
    "github-deployer|roles/run.admin"            = { sa = "github-deployer", role = "roles/run.admin" }
    "github-deployer|roles/artifactregistry.writer" = { sa = "github-deployer", role = "roles/artifactregistry.writer" }
    "github-deployer|roles/iam.serviceAccountUser"  = { sa = "github-deployer", role = "roles/iam.serviceAccountUser" }
  }
}

resource "google_project_iam_member" "bindings" {
  for_each = local.project_roles

  project = var.project_id
  role    = each.value.role
  member  = "serviceAccount:${google_service_account.accounts[each.value.sa].email}"
}

resource "google_sql_database_instance" "main" {
  project          = var.project_id
  name             = "imageservice-db"
  region           = var.region
  database_version = "POSTGRES_17"

  # Without this, a forced replacement silently destroys the database.
  deletion_protection = true

  settings {
    tier              = "db-f1-micro"
    availability_type = "ZONAL"
    disk_size         = 10

    backup_configuration {
      enabled = true
    }

    ip_configuration {
      ipv4_enabled = true
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_sql_database" "imageservice" {
  project  = var.project_id
  name     = "imageservice"
  instance = google_sql_database_instance.main.name
}

# Password set out of band: a password here is a password in plaintext state.
resource "google_sql_user" "imageservice" {
  project  = var.project_id
  name     = "imageservice"
  instance = google_sql_database_instance.main.name

  lifecycle {
    ignore_changes = [password]
  }
}

resource "google_storage_bucket" "images" {
  project                     = var.project_id
  name                        = var.bucket_name
  location                    = var.region
  uniform_bucket_level_access = false

  # The browser PUTs here directly via presigned URL. Without CORS the
  # preflight returns 200 with no allow-origin and uploads fail in-browser.
  cors {
    origin = [
      "https://httpserver-782206667056.asia-southeast1.run.app",
      "https://httpserver-kgzheofb2q-as.a.run.app",
      "http://localhost:5500",
      "http://localhost:8080",
    ]
    method          = ["PUT", "GET", "HEAD"]
    response_header = ["Content-Type"]
    max_age_seconds = 3600
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_storage_bucket_iam_member" "hmac_object_admin" {
  bucket = google_storage_bucket.images.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.accounts["gcs-hmac"].email}"
}

locals {
  # jwt-secret is authservice-only: nothing else signs or verifies tokens.
  secret_readers = {
    "database-url"     = ["authservice", "httpserver", "worker"]
    "jwt-secret"       = ["authservice"]
    "gcs-access-key"   = ["httpserver", "worker"]
    "gcs-secret-key"   = ["httpserver", "worker"]
    "temporal-api-key" = ["httpserver", "worker"]
  }

  secret_bindings = merge([
    for secret, readers in local.secret_readers : {
      for reader in readers : "${secret}|${reader}" => { secret = secret, reader = reader }
    }
  ]...)
}

# Containers only. Versions hold the values and stay out of Terraform state.
resource "google_secret_manager_secret" "secrets" {
  for_each = local.secret_readers

  project   = var.project_id
  secret_id = each.key

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "readers" {
  for_each = local.secret_bindings

  project   = var.project_id
  secret_id = google_secret_manager_secret.secrets[each.value.secret].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.accounts[each.value.reader].email}"
}

resource "google_iam_workload_identity_pool" "github" {
  project                   = var.project_id
  workload_identity_pool_id = "github"
  display_name              = "GitHub Actions"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
  }

  # Without this, any GitHub repository could mint tokens for this pool.
  attribute_condition = "assertion.repository=='${var.github_repo}'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# Lets workflows in the named repository impersonate the deployer account.
resource "google_service_account_iam_member" "github_deployer_wif" {
  service_account_id = google_service_account.accounts["github-deployer"].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${var.github_repo}"
}
