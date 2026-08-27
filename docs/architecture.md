# Architecture

The system after Phase 4 (**presigned uploads + completion signal**). Image bytes no longer
pass through `httpserver` — the client PUTs them straight to MinIO against a presigned URL,
and the workflow, which started *before* any bytes existed, is woken by a signal. Week-1
additions (auth + rate limiting) are marked; they still slot in at the HTTP edge without
changing the rest of the system.

The presigned URL exists to **reduce server load and avoid double-handling the file**: the
client uploads directly to MinIO, so the API never buffers, copies, or re-streams image bytes.

## Component diagram

```
 ┌────────────────────────────┐               ┌────────────────────────────┐
 │           Client           │──2 PUT bytes─►│ MinIO :9000                │
 └──────────────┬─────────────┘ presigned, 15m│   {id}/original            │
                │ 1 POST /uploads             │   {id}/12x12  {id}/25x25   │
                │                             └─────────────┬─────────┬────┘
                │ 3 POST /uploads/{id}/complete             │         │
                │   3' webhook  ->  /internal/s3-events     │         │
                │◄──────────────────────────────────────────┘         │
                ▼                                                     │
 ┌────────────────────────────┐                                       │
 │ httpserver :8080           │                                       │
 │ [auth + rate limit]        │                                       │
 └───┬────────────────┬───────┘                                       │
     │ start workflow │ signal "upload-complete"                      │
     ▼                ▼                                               │
 ┌────────────────────────────┐                                       │
 │ Temporal :7233             │ ◄┄┄┄ Web UI :8088                     │
 └──────────────┬─────────────┘                                       │
                │ dispatch                                            │
                ▼                                                     │
 ┌────────────────────────────┐  resize (gRPC)┌─────────────────────┐ │
 │ worker x2                  │──────────────►│  imageservice       │ │
 │  UploadWorkflow            │◄──────────────│     :50051          │ │
 │   1 wait  signal / timer   │               └─────────────────────┘ │
 │   2 Validate (sniff+size)  │ get original / put thumbnails         │
 │   3 Resize x2 (fan-out)    │◄──────────────────────────────────────┘
 │   4 Complete|Abandon|Reject│
 └──────────────┬─────────────┘
                │ status / content_type
                ▼
 ┌─────────────────────────────────────────────────────────────────────────┐
 │ Postgres :5432   uploads(status, content_type)  +  Temporal state       │
 └─────────────────────────────────────────────────────────────────────────┘
```

`POST /upload` (multipart) is **removed**, not kept alongside — two upload paths would double
the handler test surface and the second one would rot.

## Transport

```
POST /uploads               → {id, url, expires_in}    201
POST /uploads/{id}/complete → signal workflow          200
POST /internal/s3-events    → MinIO webhook → signal   204
GET  /images/{id}/status    → unchanged
```

## Flow (one upload)

```
  t=0   POST /uploads {filename, content_type}
        ├─ id.New()
        ├─ repo.Create(id, filename, contentType, "pending")
        ├─ orchestrator.Start(id)      workflow begins, blocks on Select
        ├─ store.Presign(key, PUT, 15m)
        └─ 201 {id, url, expires_in: 900}

        client ──PUT bytes──► MinIO       (httpserver sees nothing)

        POST /uploads/{id}/complete  ──┐
        ...or MinIO ObjectCreated    ──┴─► SignalWorkflow("upload-complete")

        workflow unblocks
        ├─ Validate    sniff first 512B, Head for size
        ├─ fan out     Resize x2  (unchanged)
        └─ Complete    status "complete"

  t=15m no signal ─► timer fires ─► Abandon: best-effort Delete, status "abandoned"
```

Note the ordering: the DB row and the workflow start **before** presigning. If presign fails,
the row and workflow already exist and the timer reaps them — the system self-heals rather
than orphaning state.

## Status vocabulary

```
  pending ──► validating ──► processing ──► complete
     │             │                    └──► failed
     │             └──► rejected            (resize/gRPC error)
     └──► abandoned      (bad type or size)
          (timer, no upload)
```

## Storage layer

`internal/storage` exposes an `ObjectStore` interface — `minioStore` in production, the
existing disk store retained to satisfy the interface in tests:

```go
type ObjectStore interface {
        Presign(ctx context.Context, key, method string, ttl time.Duration) (string, error)
        Get(ctx context.Context, key string) ([]byte, error)
        GetRange(ctx context.Context, key string, n int64) ([]byte, error)
        Put(ctx context.Context, key string, data []byte, contentType string) error
        Head(ctx context.Context, key string) (size int64, contentType string, err error)
        Delete(ctx context.Context, key string) error
}
```

Keys drop extensions entirely: `{id}/original`, `{id}/12x12`, `{id}/25x25`. That deletes the
`filepath.Glob` in `Path()` and dissolves the can't-know-the-extension-at-presign-time
problem. Content type moves to a new `content_type` column on `uploads`.

## Activities

| Activity | Role |
|---|---|
| `Validate` | `GetRange` first 512 bytes → `http.DetectContentType` → `Head` for size. On failure: `Delete` + status `rejected`. The existing sniff logic, relocated. |
| `Abandon` | Timer path. Best-effort `Delete`, status `abandoned`. |
| `Resize` | Now reads via `ObjectStore.Get` instead of `Store.Read`; writes via `Put`. |
| `Complete` / `Fail` | Unchanged. |

## Two things worth naming

**Duplicate signals need no explicit guard.** `Select` returns once. A second
`upload-complete` — from the webhook, arriving after the client callback already won — lands
in the channel buffer and is simply never read. Temporal doesn't error on it. The dedup you'd
expect to write turns out to be structural.

**The real idempotency problem is at the HTTP edge, not in the workflow.** If the client calls
`/complete` twice and the workflow has already finished, `SignalWorkflow` returns a `NotFound`
error. That must map to `200`, not `500` — the caller's intent was satisfied, just earlier.

`cancelTimer()` matters too: without it the timer stays live in history until it fires,
leaving 15-minute-long zombie timers on every successful upload.

## Legend
- Solid arrows = request/data flow; dotted (`┄`) = observation only (Web UI).
- Numbered edges `1 2 3` are the client's happy path, in order; `3'` is the webhook
  alternative to `3` — either one signals the workflow, whichever arrives first.
- Transports: **HTTP** client→httpserver · **S3/HTTP** client→MinIO and worker→MinIO ·
  **gRPC** httpserver→Temporal and worker→imageservice · **SQL** →Postgres.
- The Postgres band is shared state: `httpserver` also writes the `pending` row and reads
  status from it; only the worker's write is drawn, to keep the spine readable.
