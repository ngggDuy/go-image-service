# Import blocks adopt the resources that already exist in the project. They are
# declarative: `terraform plan` shows what each one will bind to, and nothing is
# created for an address that appears here.
#
# Once the first apply succeeds and state is populated, this file can be
# deleted -- the imports are a one-time reconciliation, not ongoing config.

import {
  to = google_project_service.enabled["artifactregistry.googleapis.com"]
  id = "img-svc-train/artifactregistry.googleapis.com"
}

import {
  to = google_project_service.enabled["iam.googleapis.com"]
  id = "img-svc-train/iam.googleapis.com"
}

import {
  to = google_project_service.enabled["iamcredentials.googleapis.com"]
  id = "img-svc-train/iamcredentials.googleapis.com"
}

import {
  to = google_project_service.enabled["run.googleapis.com"]
  id = "img-svc-train/run.googleapis.com"
}

import {
  to = google_project_service.enabled["secretmanager.googleapis.com"]
  id = "img-svc-train/secretmanager.googleapis.com"
}

import {
  to = google_project_service.enabled["sqladmin.googleapis.com"]
  id = "img-svc-train/sqladmin.googleapis.com"
}

import {
  to = google_project_service.enabled["storage.googleapis.com"]
  id = "img-svc-train/storage.googleapis.com"
}

import {
  to = google_artifact_registry_repository.services
  id = "projects/img-svc-train/locations/asia-southeast1/repositories/services"
}

import {
  to = google_sql_database_instance.main
  id = "projects/img-svc-train/instances/imageservice-db"
}

import {
  to = google_sql_database.imageservice
  id = "projects/img-svc-train/instances/imageservice-db/databases/imageservice"
}

import {
  to = google_sql_user.imageservice
  id = "img-svc-train/imageservice-db/imageservice"
}

import {
  to = google_storage_bucket.images
  id = "img-svc-train/img-svc-train-images"
}

import {
  to = google_service_account.accounts["imageservice"]
  id = "projects/img-svc-train/serviceAccounts/imageservice-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_service_account.accounts["authservice"]
  id = "projects/img-svc-train/serviceAccounts/authservice-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_service_account.accounts["httpserver"]
  id = "projects/img-svc-train/serviceAccounts/httpserver-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_service_account.accounts["worker"]
  id = "projects/img-svc-train/serviceAccounts/worker-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_service_account.accounts["github-deployer"]
  id = "projects/img-svc-train/serviceAccounts/github-deployer-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_service_account.accounts["gcs-hmac"]
  id = "projects/img-svc-train/serviceAccounts/gcs-hmac-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_storage_bucket_iam_member.hmac_object_admin
  id = "b/img-svc-train-images roles/storage.objectAdmin serviceAccount:gcs-hmac-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_project_iam_member.bindings["authservice|roles/cloudsql.client"]
  id = "img-svc-train roles/cloudsql.client serviceAccount:authservice-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_project_iam_member.bindings["httpserver|roles/cloudsql.client"]
  id = "img-svc-train roles/cloudsql.client serviceAccount:httpserver-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_project_iam_member.bindings["worker|roles/cloudsql.client"]
  id = "img-svc-train roles/cloudsql.client serviceAccount:worker-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_project_iam_member.bindings["github-deployer|roles/run.admin"]
  id = "img-svc-train roles/run.admin serviceAccount:github-deployer-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_project_iam_member.bindings["github-deployer|roles/artifactregistry.writer"]
  id = "img-svc-train roles/artifactregistry.writer serviceAccount:github-deployer-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_project_iam_member.bindings["github-deployer|roles/iam.serviceAccountUser"]
  id = "img-svc-train roles/iam.serviceAccountUser serviceAccount:github-deployer-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret.secrets["database-url"]
  id = "projects/img-svc-train/secrets/database-url"
}

import {
  to = google_secret_manager_secret.secrets["jwt-secret"]
  id = "projects/img-svc-train/secrets/jwt-secret"
}

import {
  to = google_secret_manager_secret.secrets["gcs-access-key"]
  id = "projects/img-svc-train/secrets/gcs-access-key"
}

import {
  to = google_secret_manager_secret.secrets["gcs-secret-key"]
  id = "projects/img-svc-train/secrets/gcs-secret-key"
}

import {
  to = google_secret_manager_secret.secrets["temporal-api-key"]
  id = "projects/img-svc-train/secrets/temporal-api-key"
}

import {
  to = google_secret_manager_secret_iam_member.readers["database-url|authservice"]
  id = "projects/img-svc-train/secrets/database-url roles/secretmanager.secretAccessor serviceAccount:authservice-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["database-url|httpserver"]
  id = "projects/img-svc-train/secrets/database-url roles/secretmanager.secretAccessor serviceAccount:httpserver-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["database-url|worker"]
  id = "projects/img-svc-train/secrets/database-url roles/secretmanager.secretAccessor serviceAccount:worker-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["jwt-secret|authservice"]
  id = "projects/img-svc-train/secrets/jwt-secret roles/secretmanager.secretAccessor serviceAccount:authservice-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["gcs-access-key|httpserver"]
  id = "projects/img-svc-train/secrets/gcs-access-key roles/secretmanager.secretAccessor serviceAccount:httpserver-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["gcs-access-key|worker"]
  id = "projects/img-svc-train/secrets/gcs-access-key roles/secretmanager.secretAccessor serviceAccount:worker-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["gcs-secret-key|httpserver"]
  id = "projects/img-svc-train/secrets/gcs-secret-key roles/secretmanager.secretAccessor serviceAccount:httpserver-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["gcs-secret-key|worker"]
  id = "projects/img-svc-train/secrets/gcs-secret-key roles/secretmanager.secretAccessor serviceAccount:worker-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["temporal-api-key|httpserver"]
  id = "projects/img-svc-train/secrets/temporal-api-key roles/secretmanager.secretAccessor serviceAccount:httpserver-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_secret_manager_secret_iam_member.readers["temporal-api-key|worker"]
  id = "projects/img-svc-train/secrets/temporal-api-key roles/secretmanager.secretAccessor serviceAccount:worker-sa@img-svc-train.iam.gserviceaccount.com"
}

import {
  to = google_iam_workload_identity_pool.github
  id = "projects/img-svc-train/locations/global/workloadIdentityPools/github"
}

import {
  to = google_iam_workload_identity_pool_provider.github
  id = "projects/img-svc-train/locations/global/workloadIdentityPools/github/providers/github-provider"
}

import {
  to = google_service_account_iam_member.github_deployer_wif
  id = "projects/img-svc-train/serviceAccounts/github-deployer-sa@img-svc-train.iam.gserviceaccount.com roles/iam.workloadIdentityUser principalSet://iam.googleapis.com/projects/782206667056/locations/global/workloadIdentityPools/github/attribute.repository/ngggDuy/go-image-service"
}
