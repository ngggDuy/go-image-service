# go-image-service

An image upload and processing service. Upload an image and it's resized to
12×12 and 25×25; fetch the original or any resized version back by URL.

## Architecture

Three services, run together with Docker Compose:

| Service | Role | Port |
|---------|------|------|
| `httpserver` | HTTP API — accepts uploads, stores files, writes metadata | `8080` (host) |
| `imageservice` | gRPC service that resizes image bytes (stateless) | `50051` (internal only) |
| `postgres` | Stores upload metadata | `5432` (host) |

Flow: client → **HTTP** → `httpserver` → **gRPC** → `imageservice` (resize).
`httpserver` stores the image files on disk and the metadata in Postgres.

## Prerequisites

- **Docker** and **Docker Compose**.

## Run

```bash
git clone https://github.com/ngggDuy/go-image-service.git
cd go-image-service
docker compose up --build
```

Wait until Postgres reports healthy and you see `image service listening on :50051`.
The API is then available at `http://localhost:8080`.

Run in background with: 
`docker compose up -d --build`

Follow logs with:
`docker compose logs -f httpserver imageservice`.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/upload` | Upload an image (`multipart/form-data`, field name `image`, JPEG/PNG, max 10 MB). Returns `{"id":"..."}`. |
| `GET` | `/images/{id}?size=` | Fetch an image. `size` is one of `original`, `12x12`, `25x25`. Returns the image bytes. |
| `GET` | `/health` | Liveness check. Returns `{"status":"ok"}`. |


## Inspect the database

```bash
docker compose exec postgres psql -U imageservice -d imageservice -c "SELECT * FROM uploads;"
```

Or connect any SQL client to `localhost:5432` — database `imageservice`, user
`imageservice`, password `secret`.

## Stop

```bash
docker compose down        # stop and remove containers
docker compose down -v     # also delete the database + uploads volumes
```

