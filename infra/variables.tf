variable "project_id" {
  type    = string
  default = "img-svc-train"
}

variable "project_number" {
  type    = string
  default = "782206667056"
}

variable "region" {
  type    = string
  default = "asia-southeast1"
}

variable "github_repo" {
  description = "owner/repo allowed to impersonate the deployer service account"
  type        = string
  default     = "ngggDuy/go-image-service"
}

variable "bucket_name" {
  type    = string
  default = "img-svc-train-images"
}
