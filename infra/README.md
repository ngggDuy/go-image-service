# Terraform: foundational infrastructure

State lives in `gs://img-svc-train-tfstate`, prefix `foundation`. Run from
Cloud Shell, where Terraform is preinstalled:

```bash
cd infra
terraform init
terraform plan
```

A clean tree reports `No changes.` Anything else means something drifted —
usually a console edit someone made by hand.

## Scope

Terraform owns what holds state or grants access, and what should not change on
every merge:

- Cloud SQL instance, database, user
- GCS bucket and its IAM
- Service accounts and project-level IAM
- Secret Manager secret *containers* and their IAM
- Workload Identity Federation pool and provider
- Artifact Registry repository
- Enabled APIs

**Cloud Run services and the worker pool are deliberately excluded.**
`.github/workflows/deploy.yml` deploys those on every push to `main` with a new
image tag. If both systems managed them, every apply would revert the running
image and every deploy would surface as drift.

**Secret values are also excluded.** Only the containers are managed; versions
are set with `gcloud secrets versions add`. A value passed through Terraform is
a value written to state in plaintext.

## Safety

`prevent_destroy` is set on the SQL instance and the bucket, so Terraform
refuses to destroy them or apply any change forcing replacement. The instance
additionally has `deletion_protection_enabled`, which blocks deletion via the
console and gcloud too.

## Applying

Always apply a saved plan, so what you reviewed is what runs:

```bash
terraform plan -out=tfplan
terraform apply tfplan
```
