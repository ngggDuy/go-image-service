# go-image-service

Upload an image and it's resized to 12×12 and 25×25; fetch the original or any
resized version back by URL.

## API

Base URL: `http://localhost:8080`

The full, formal contract is in **[`openapi.yaml`](./openapi.yaml)**.

| Method | Path | Request | Success | Errors |
|--------|------|---------|---------|--------|
| `POST` | `/uploads` | `{"filename":"...","content_type":"image/jpeg"}` | `201` → `{"id":"...","url":"...","expires_in":900}` | `400`, `500` |
| `POST` | `/uploads/{id}/complete` | — | `200` (idempotent) | `400`, `404`, `500` |
| `GET` | `/images/{id}/status` | — | `200` → `{"id":"...","status":"..."}` | `400`, `404` |
| `GET` | `/images/{id}?size=` | `size` = `original` \| `12x12` \| `25x25` | `200` → image bytes | `400`, `404` |
| `GET` | `/health` | — | `200` → `{"status":"ok"}` | — |

> **Breaking change:** the multipart `POST /upload` is **removed**, replaced by the presigned
> flow below. There is no compatibility shim.

Uploads are **presigned and asynchronous**. `POST /uploads` creates the record, starts the
Temporal workflow (which immediately blocks waiting for a completion signal), and returns a
presigned PUT URL valid for 15 minutes. The client PUTs the bytes **directly to object
storage** — they never pass through this service — then calls `POST /uploads/{id}/complete`.
A MinIO `ObjectCreated` webhook signals the same workflow, so an upload still progresses if
the client disappears after the PUT; whichever signal arrives first wins and the duplicate is
harmlessly ignored.

Poll `GET /images/{id}/status` until it reads `complete`, then fetch via
`/images/{id}?size={original|12x12|25x25}`. Statuses:

`pending` → `validating` → `processing` → `complete` | `failed` | `rejected` | `abandoned`

If the client never signals, a 15-minute timer fires: the object is deleted and the upload is
marked `abandoned`.

### Examples

```bash
# 1. negotiate — returns {"id":"...","url":"...","expires_in":900}
curl -s -X POST http://localhost:8080/uploads \
  -H 'Content-Type: application/json' \
  -d '{"filename":"photo.jpg","content_type":"image/jpeg"}'

# 2. PUT the bytes straight to storage using the presigned url from above
curl -s -X PUT --upload-file /path/to/photo.jpg \
  -H 'Content-Type: image/jpeg' "THE_PRESIGNED_URL"

# 3. signal completion
curl -s -X POST http://localhost:8080/uploads/THE_ID/complete

# 4. poll until "complete"
curl -s http://localhost:8080/images/THE_ID/status

# 5. fetch a resized version
curl -s "http://localhost:8080/images/THE_ID?size=12x12" --output thumb.jpg

# health
curl -s http://localhost:8080/health
```

## Run

**Prerequisites:** Docker and Docker Compose.

```bash
git clone https://github.com/ngggDuy/go-image-service.git
cd go-image-service
make up            # build and start the full stack
```

Wait until Postgres reports healthy and you see `image service listening on :50051`.
The API is then available at `http://localhost:8080`.

Common commands (run `make` with no argument to list them all):

| Command | Does |
|---------|------|
| `make up` | build + start (foreground) |
| `make up-d` | build + start (background) |
| `make logs` | follow the Go services' logs |
| `make ps` | container status |
| `make db` | open a `psql` shell |
| `make down` | stop containers |
| `make clean` | stop **and** wipe volumes (fresh DB) |

## How it works

Services, run together with Docker Compose:

| Service | Role | Port |
|---------|------|------|
| `httpserver` | HTTP API — negotiates presigned uploads, signals the workflow, serves images | `8080` (host) |
| `worker` (×2) | Temporal workers that run the validate and resize activities | internal |
| `imageservice` | gRPC service that resizes image bytes (stateless) | `50051` (internal) |
| `temporal` | Temporal cluster (orchestration engine) | `7233` (internal) |
| `temporal-ui` | Temporal Web UI | `8088` (host) |
| `minio` | S3-compatible object storage for originals and thumbnails | `9000` (host) |
| `postgres` | Upload metadata (and Temporal's own database) | `5432` (host) |

Flow: `POST /uploads` starts a **Temporal workflow** *before any bytes exist* — its first act
is to wait, on a `Select` over the `upload-complete` signal and a 15-minute timer. The client
PUTs the bytes straight to MinIO; the completion signal (from the client or the MinIO webhook)
unblocks the workflow, which validates the object, fans out the two resizes as **parallel
activities** across the workers — each calling `imageservice` over **gRPC** — and marks the
upload `complete`. If the timer wins instead, the object is deleted and the upload is marked
`abandoned`. Watch it live in the Temporal UI at **http://localhost:8088**. Full explanation in
[`docs/temporal.md`](./docs/temporal.md), architecture in
[`docs/architecture.md`](./docs/architecture.md).

Design notes:
- The **image service is stateless** — bytes in, resized bytes out. All state
  (objects, database) is owned by the HTTP server and the workers.
- **Bytes bypass the API** — clients PUT to and (optionally) GET from object storage
  against presigned URLs; `httpserver` never buffers an upload.
- **Objects in MinIO, metadata in Postgres** — the DB holds records about uploads, not
  the image bytes themselves. Keys are extension-less: `{id}/original`, `{id}/12x12`,
  `{id}/25x25`; the content type lives in the `content_type` column.
- **gRPC internally, HTTP externally** — clients speak plain HTTP; the two Go
  services speak gRPC to each other.
- IDs are random (`crypto/rand`) and therefore unguessable — the only access guard,
  since there is no authentication.

### Inspect the database

```bash
docker compose exec postgres psql -U imageservice -d imageservice -c "SELECT * FROM uploads;"
```

Or connect any SQL client to `localhost:5432` — database `imageservice`, user
`imageservice`, password `secret`.

## Development

Only needed if you change the code or the `.proto` contract.

- **Regenerate gRPC code** after editing `proto/imageprocess.proto`: `make proto`
  (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, with `$(go env GOPATH)/bin`
  on your `PATH`).
- **Run without Docker:** bring up the backing services
  (`docker compose up -d postgres temporal imageservice`), then in separate terminals run
  `go run ./cmd/worker` and `go run ./cmd/httpserver`.
