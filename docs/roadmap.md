# go-image-service — 1-Month Growth Roadmap

Goal: evolve the project from a working demo into a production-shaped backend, going
deep on **gRPC** and **Temporal**, and adding **banking-relevant** concerns (auth,
idempotency, encryption, audit, reconciliation). Includes **ConnectRPC** and **Terraform**.

## How to use this (read first)
This is deliberately ambitious. **Depth beats breadth** — it's better to implement
client-streaming + child workflows + tracing *well* than to rush all 30 items shallowly.
Each week has a **Core spine** (do these) and **Stretch** (if time). Weeks are ordered by
dependency: identity → gRPC → Temporal → production. Ship each item on its own branch with
tests.

Banking framing worth knowing: Temporal is widely used in **fintech/payments** precisely
for durable execution, sagas, idempotency, and auditability. Every Temporal pattern below
(saga, idempotency keys, encrypted payloads, reconciliation) transfers directly to money
movement — good interview material.

---

## Week 1 — Identity, multi-tenancy & guardrails
*Foundations almost everything else needs.*

**Core spine**
- **AuthN**: JWT (or API keys) + a login endpoint; middleware that resolves the caller.
- **Per-user ownership**: add `user_id` to `uploads`; users only see/fetch their own images.
- **List + cursor pagination**: `GET /images` for the current user.
- **Rate limiting**: token-bucket HTTP middleware (per-user/IP) — your "1000 users × 100
  images" scenario — plus per-user **upload quotas** (DB-backed).
- **Idempotency keys** 🏦: accept an `Idempotency-Key` header; dedupe retried uploads. Ties
  neatly to Temporal's workflow-id dedupe you already use (`upload-{id}`).

**Stretch**
- **RBAC** 🏦 (user vs admin roles) + an admin "list all uploads" view.
- Graceful shutdown (drain in-flight work on SIGTERM).

*Skills: middleware, auth, access control, DB modeling, pagination.*

---

## Week 2 — gRPC mastery + ConnectRPC

**Core spine**
- **gRPC interceptors**: unary + stream interceptors for logging, metrics, request IDs, and
  auth propagation. (Interceptors = gRPC's middleware.)
- **Client-streaming upload**: replace the single 10 MB message (the one that made you bump
  `MaxRecvMsgSize`) with a chunked client stream — the correct pattern for large payloads.
- **Deadlines & cancellation propagation**: set a call deadline; watch it cancel the resize.
- **Migrate to ConnectRPC** (your `go-doc.md` Phase 3.5): same protos via `buf`, but
  browser-callable and HTTP/JSON-friendly. Big, résumé-worthy modernization.

**Stretch**
- **Server-streaming download**; **multiple `imageservice` replicas + client-side load
  balancing**; **mTLS** 🏦 between services (replace `insecure` creds); rich error details.

*Skills: streaming RPCs, interceptors, deadlines, buf/ConnectRPC, TLS, load balancing.*

---

## Week 3 — Temporal mastery (the deep week)

**Core spine**
- **Bulk upload → parent + child workflows**: "upload 100 images" = a parent workflow
  spawning a child workflow per image, with **bounded concurrency** (back-pressure).
- **Download-all-as-zip**: a long-running workflow (gather → zip activity → store → notify).
- **Activity heartbeats + cancellation** for very large/slow images ("what if it takes
  forever?").
- **Workflow versioning** (`workflow.GetVersion`): change the workflow safely while old runs
  are in flight — the skill that separates production Temporal from toy Temporal.

**Stretch (several are banking gold)**
- **Encrypted payload Data Converter** 🏦: encrypt everything Temporal writes to history.
  This is *the* fintech Temporal feature — sensitive data never sits in plaintext in the
  event store.
- **Signals & queries**: push live progress / cancel a run (upgrades your status polling).
- **Scheduled cleanup + reconciliation workflow** 🏦: a cron workflow that deletes expired
  uploads and reconciles DB rows vs. files on disk (finds orphans/missing) — banks live on
  reconciliation.
- **Saga / compensation**: if one resize permanently fails, compensate (delete partial
  results, mark failed).
- **Search attributes** (`userId`, `status`) for querying in the UI/CLI.
- **Data retention / right-to-erasure** 🏦: scheduled workflow enforcing a retention policy.

*Skills: child workflows, heartbeats, versioning, signals/queries, schedules, sagas, data
converters.*

---

## Week 4 — Production: observability, storage, IaC, CI/CD
*This is your `go-doc.md` Phase 4, expanded.*

**Core spine**
- **Distributed tracing (OpenTelemetry)** ⭐: propagate one trace across HTTP → Temporal →
  gRPC and view it in Jaeger. Nothing teaches distributed systems faster than *seeing* the
  full path.
- **Object storage (MinIO/S3)** + **presigned URLs**: replace local disk / the shared
  volume; the real-world pattern.
- **CI (GitHub Actions)**: lint → test → build; **integration tests** (testcontainers) and a
  **load test** (k6) for the rate-limit/scale scenarios.

**Stretch**
- **Prometheus + Grafana** metrics (upload rate, resize latency, queue depth) + Temporal
  metrics.
- **Structured logging** (slog) + request IDs; **immutable audit log** 🏦 (who did what,
  when — Temporal history is already one; add app-level audit for HTTP actions).
- **Encryption at rest** 🏦 for stored images; **secrets management** 🏦 (Vault or at least
  no hardcoded `secret`).
- **DB migrations** (goose/atlas) + **sqlc** (type-safe queries, replaces hand-written SQL).
- **Terraform** (see below).

*Skills: tracing, metrics, object storage, CI/CD, IaC, migrations.*

---

## Terraform (where it fits — you flagged it as a stretch, and it is)
Two honest options, easiest first:
1. **Docker provider (local IaC)**: re-express your `docker-compose` stack as Terraform HCL
   using the `kreuzwerker/docker` provider. Teaches HCL, providers, resources, and **state**
   without any cloud cost. Best learning-per-dollar.
2. **Cloud provisioning (real Phase-4+)**: Terraform an AWS/GCP deployment — S3 bucket
   (object storage), RDS Postgres, a container runtime (ECS/Cloud Run), and a **Temporal
   Cloud** namespace. This is what "Terraform for this project" looks like in production, but
   it costs money and time. Do this only if Week 4 core is done.

## Banking-relevant themes (consolidated)
🏦 Idempotency keys · RBAC · mTLS · **encrypted Temporal payloads (Data Converter)** ·
audit trail / immutable logging · reconciliation workflow · data retention / right-to-erasure ·
encryption at rest · secrets management. Pitch: these are the exact controls a bank applies
to money-movement pipelines — you're practicing them on images.

## Suggested definition of done for each item
Branch → implement → unit/integration tests → update `openapi.yaml` (if API changes) →
update the relevant doc → merge. Keep coverage on logic packages high, as you have been.
