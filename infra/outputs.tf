output "sql_connection_name" {
  value = google_sql_database_instance.main.connection_name
}

output "wif_provider" {
  description = "Value for the GCP_WIF_PROVIDER repository variable."
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "deployer_service_account" {
  description = "Value for the GCP_DEPLOYER_SA repository variable."
  value       = google_service_account.accounts["github-deployer"].email
}

output "bucket" {
  value = google_storage_bucket.images.name
}
