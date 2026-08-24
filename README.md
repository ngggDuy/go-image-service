# go-image-service

Upload an image and it's resized to 12×12 and 25×25; fetch the original or any
resized version back by URL.

## API

Base URL: `http://localhost:8080`

The full, formal contract is in **[`openapi.yaml`](./openapi.yaml)**.

| Method | Path | Request | Success | Errors |
|--------|------|---------|---------|--------|
| `POST` | `/upload` | `multipart/form-data`, field **`image`** (JPEG/PNG, ≤ 10 MB) | `202` → `{"id":"..."}` | `400`, `413`, `500` |
| `GET` | `/images/{id}/status` | — | `200` → `{"id":"...","status":"..."}` | `400`, `404` |
| `GET` | `/images/{id}?size=` | `size` = `original` \| `12x12` \| `25x25` | `200` → image bytes | `400`, `404` |
| `GET` | `/health` | — | `200` → `{"status":"ok"}` | — |

Upload is **asynchronous**: `POST /upload` returns `202 Accepted` with the `id`, then a
Temporal workflow resizes the image in the background. Poll `GET /images/{id}/status` until it
reads `complete` (`processing` → `complete` / `failed`), then fetch images via
`/images/{id}?size={original|12x12|25x25}`.

### Examples

```bash
# upload (returns {"id":"..."})
curl -s -X POST -F "image=@/path/to/photo.jpg" http://localhost:8080/upload

# fetch a resized version (use the id from above)
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
| `httpserver` | HTTP API — stores the original, starts the resize workflow, serves images | `8080` (host) |
| `worker` (×2) | Temporal workers that run the resize activities | internal |
| `imageservice` | gRPC service that resizes image bytes (stateless) | `50051` (internal) |
| `temporal` | Temporal cluster (orchestration engine) | `7233` (internal) |
| `temporal-ui` | Temporal Web UI | `8088` (host) |
| `postgres` | Upload metadata (and Temporal's own database) | `5432` (host) |

Flow: `POST /upload` saves the original and starts a **Temporal workflow**, which fans out
the two resizes as **parallel activities** across the workers; each activity calls
`imageservice` over **gRPC** and stores the result, then the workflow marks the upload
`complete`. Watch it live in the Temporal UI at **http://localhost:8088**. Full explanation in
[`docs/temporal.md`](./docs/temporal.md).

Design notes:
- The **image service is stateless** — bytes in, resized bytes out. All state
  (files, database) is owned by the HTTP server.
- **Files on disk, metadata in Postgres** — the DB holds records about uploads, not
  the image bytes themselves.
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
