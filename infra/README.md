# Terraform: foundational infrastructure

## Scope

Terraform owns the things that hold state or grant access, and that should not
change on every merge:

- Cloud SQL instance, database, user
- GCS bucket and its IAM
- Service accounts and project-level IAM
- Secret Manager secret *containers* and their IAM
- Workload Identity Federation pool and provider
- Artifact Registry repository
- Enabled APIs

**Terraform does not own the Cloud Run services or the worker pool.** Those are
deployed by `.github/workflows/deploy.yml` on every push to `main`, with a new
image tag each time. If both systems managed them, every `terraform apply`
would revert to whatever image was current when the config was written, and
every deploy would show up as drift.

Terraform also does not own **secret values**. Only the containers are managed;
versions are set out of band with `gcloud secrets versions add`. A value passed
through Terraform is a value written to state in plaintext.

## First run

Terraform needs a state bucket before it can store state, so create that one
resource by hand:

```bash
gcloud storage buckets create gs://img-svc-train-tfstate \
  --project=img-svc-train --location=asia-southeast1 \
  --uniform-bucket-level-access
gcloud storage buckets update gs://img-svc-train-tfstate --versioning
```

Then, from Cloud Shell:

```bash
git clone https://github.com/ngggDuy/go-image-service.git
cd go-image-service/infra
terraform init
terraform plan
```

## Reading that first plan

Every resource here already exists, so a correct plan reports **42 to import
and 0 to add**. Read it in this order:

1. **"will be imported"** — expected, one per `import` block.
2. **"must be replaced"** — stop. Replacing a real resource destroys it. This
   means a config attribute does not match reality; fix the config, not the
   infrastructure.
3. **"will be created"** — an import block is missing or its id is wrong.
   Terraform is about to create a duplicate of something that exists.
4. **"will be updated in place"** — read each one. Some are intentional (see
   below); anything else is a mismatch to reconcile.

Two updates are deliberate, and both close real gaps:

- `backup_configuration.enabled: false -> true` — the instance currently has
  **no backups**.
- `deletion_protection: false -> true` — the instance can currently be deleted
  by a single command.

Once the apply succeeds, `imports.tf` has done its job and can be deleted.

## Safety

`prevent_destroy` is set on the SQL instance and the bucket. Terraform will
refuse to destroy them, and refuse any change that forces replacement, until
someone removes that line deliberately.
