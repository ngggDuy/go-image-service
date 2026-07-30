# go-image-service — System Description

## Overview

An image upload and processing service.

Users upload an image, the server resizes it to multiple dimensions (12x12, 25x25). Users can fetch the original and resized images via URL.

## Architecture (final state)

```
[Frontend] → HTTP → [Selfie Server] → gRPC → [ImageProcess Service]
                           ↓                     ├── Resize 12x12
                    [Temporal Worker]              └── Resize 25x25
                      ├── Activity: Resize(12x12)
                      └── Activity: Resize(25x25)
```

## Components

| Component | Role |
|-----------|------|
| **Selfie Server** | Accepts image uploads, stores originals, tracks processing status |
| **ImageProcess Service** | Resizes images to requested dimensions using Go's `image` stdlib |
| **Temporal Worker** | Orchestrates resize operations in parallel, aggregates results |

## Data

- **Postgres** — stores image metadata
  - **uploads** — upload metadata (id, filename, status, created_at)
  - **results** — processing results (id, upload_id, width, height, file_path)
- **File storage** — original and resized images on disk

## Phases

### Phase 1 — Go Fundamentals + Web Server

Set up a basic HTTP server and learn core Go concepts:
- Variables, types, constants
- Structs, methods, interfaces
- Slices, maps, pointers
- Error handling (`error`, wrapping, `errors.Is`/`As`)
- Packages and modules (`go mod`)
- Functions, control flow, iteration
- File I/O, byte slices, image handling basics

**Deliverable:** A running HTTP server with basic endpoints (e.g. `POST /upload`, `GET /images/{id}`, `GET /health`). Accepts file uploads, serves original and resized images via URL. Runnable via `go run`. README documents how to build and run.

### Phase 2 — Backend Service + gRPC + Postgres

- Define the image processing logic as a separate backend service with clear function signatures
- Connect the web server to the backend service via gRPC (protobuf definitions, codegen)
- Web server serves real HTTP endpoints externally, calls backend service internally via gRPC
- Add Postgres for upload/result metadata persistence
- Add unit tests for image processing logic and gRPC handlers

**Deliverable:** Two services running together. HTTP → gRPC → Postgres + file storage. Unit tests passing. `docker-compose` for Postgres.

### Phase 3 — Temporal

- Replace sequential resize calls with Temporal workflow
- Fan-out: `Resize(12x12)` and `Resize(25x25)` run as parallel activities
- Aggregate results on completion
- Timeout handling for large images

**Deliverable:** Temporal worker as a separate binary. `docker-compose` includes Temporal server + UI. Resize operations orchestrated in parallel by Temporal.

### Phase 3.5 (TBD) — ConnectRPC

- Migrate from standard gRPC to ConnectRPC

### Phase 4 — Deployment + CI + Integration Tests

- Dockerfile (multi-stage build) for server and worker
- `docker-compose` for full local environment
- GitHub Actions pipeline: lint → test → build
- Integration/blackbox tests against running services
- README with full setup, run, test, and deploy instructions

**Deliverable:** CI green. Full stack runs via `docker-compose up`. Integration tests pass.
 