# Phase 1 API Endpoints

This service lets a user upload a selfie image, have it resized to preset
dimensions, and fetch the original or resized versions back by URL. These are
the endpoints suggested in the Phase 1 deliverable.

**Key assumption for Phase 1:** processing is **synchronous** 
---

## 1. `POST /upload`

Accept an image, store the original, and resize it to 12×12 and 25×25 before
responding.

**Request**

| | |
|---|---|
| Body | `multipart/form-data` |
| Field name | `image` |
| Accepted formats | JPEG, PNG |
| Max size | 10 MB |

**Success — `201 Created`**

Returns JSON with the generated `id`. The client builds the image URLs from
this id using the known scheme `/images/{id}?size={original|12x12|25x25}`.

```json
{
  "id": "a1b2c3"
}
```

**Errors**

| Code | When |
|------|------|
| `400 Bad Request` | No file, or an unsupported file type |
| `413 Content Too Large` | File exceeds the 10 MB limit |
| `500 Internal Server Error` | Server failed while storing or resizing a valid image |

---

## 2. `GET /images/{id}`

Fetch one specific stored image by its `id`. Which version to return is selected
with a `size` **query parameter**.

**Request**

| | |
|---|---|
| Path param | `id` — the id returned by `POST /upload` |
| Query param | `size` — one of `original`, `12x12`, `25x25` |
| Body | none |

Example: `GET /images/a1b2c3?size=12x12`

**Success — `200 OK`**

Returns the raw image bytes (with the appropriate `Content-Type`, e.g.
`image/jpeg`).

**Errors**

| Code | When |
|------|------|
| `400 Bad Request` | `size` is missing or not one of the allowed values |
| `404 Not Found` | No image exists for the given `id` |
| `500 Internal Server Error` | Server failed while reading a file that should exist |

---

## 3. `GET /health`

Report that the server is alive.

**Request:** none.

**Success — `200 OK`** with a small status body (e.g. `{"status":"ok"}`).

