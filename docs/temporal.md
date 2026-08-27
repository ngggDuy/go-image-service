# Temporal — what it is and how we use it

A primer for this project. Read top-to-bottom; the last section walks through our
actual `UploadWorkflow`.

## 1. The one-line mental model (and how it differs from Airflow)

You said "it's kind of like Apache Airflow that orchestrates a pipeline." That's a
reasonable first guess, but it's the wrong mental model, and the difference matters:

- **Airflow** is a *scheduler* for DAGs of tasks. You describe a graph; it runs tasks
  on a schedule and tracks their state externally.
- **Temporal** is **durable execution**. You write an *ordinary function* — the
  **workflow** — in normal Go (loops, ifs, variables), and Temporal makes that function
  **survive process crashes, restarts, and machine failures**, resuming exactly where it
  left off. It's less "pipeline scheduler," more "your function, made crash-proof."

So the unit isn't a DAG node — it's a *program* that can run for seconds or months and
never loses its place.

## 2. The pieces

| Piece | What it is |
|-------|-----------|
| **Temporal Cluster (Server)** | The orchestration engine. It hands work to your workers, enforces timeouts/retries, and **persists every step** to a database (Postgres, in our compose). It is *not* just a database — it's the brain. |
| **Database** | Where the Cluster stores each workflow's **event history**. |
| **Client** | Your app code that **starts** workflows (`ExecuteWorkflow`) and queries them. Our HTTP server is a client. |
| **Worker** | A process you run that **executes your workflow and activity code**. It long-polls a task queue for work. Our `cmd/worker` is a worker. |
| **Task Queue** | A named mailbox (ours: `image-resize`). Clients put work on it; workers pull from it. Multiple workers on the same queue share the load. |
| **Workflow** | The orchestration function. **Deterministic**, no direct I/O. |
| **Activity** | A single unit of real work (I/O allowed): call a service, read a file, write to the DB. |
| **Web UI** | A dashboard (ours: `http://localhost:8088`) showing every workflow, its live progress, inputs, retries, and history. |

## 3. The core magic: event sourcing + deterministic replay

This is the part your notes got *almost* right ("states are saved so it doesn't
change... idempotent?"). Here's the precise version:

- As a workflow runs, every meaningful step — "started activity X", "activity X returned
  Y", "timer fired" — is appended to an **event history** in the Cluster's database.
- If the worker crashes mid-workflow, another worker picks it up and **replays the event
  history from the start**, re-running the workflow *code* to rebuild its in-memory state.
- **Crucially, replay does NOT re-run activities that already completed.** When replay
  reaches "activity X returned Y", it just feeds `Y` back from history. So the expensive/
  side-effecting work happens **once**, even across crashes.

Two consequences you must remember:

1. **Workflows must be DETERMINISTIC.** Because the code is re-run during replay, it must
   make the *same decisions* every time. So inside a workflow: **no** `time.Now()`, **no**
   `rand`, **no** direct network/DB/file calls, **no** goroutines/`select` on real
   channels. Use the Temporal equivalents (`workflow.Now`, `workflow.Sleep`,
   `workflow.ExecuteActivity`). If a workflow is non-deterministic, replay diverges and
   Temporal errors out.
2. **Activities should be IDEMPOTENT.** An activity *can* be retried (that's the point), so
   running it twice must be safe. Our `Resize` activity overwrites the same output file, so
   re-running it is harmless — idempotent by construction.

So "idempotent" applies to **activities**; the *workflow* gets its safety from
**deterministic replay of an event history**, not from idempotency.

## 4. Retries and timeouts (why Temporal shines for slow/large images)

Every activity runs under an `ActivityOptions` policy. Ours:

- **`StartToCloseTimeout: 2m`** — a single attempt may take up to 2 minutes (generous for a
  huge image) before it's considered timed out.
- **`RetryPolicy` (3 attempts, exponential backoff)** — if an attempt fails or times out,
  Temporal automatically retries it, backing off between tries. You write zero retry code.

Other timeouts worth knowing: **ScheduleToClose** (total budget across all retries) and
**Heartbeat** (for long activities that must periodically prove they're alive).

## 5. Fan-out / fan-in (our exact pattern)

The workflow starts **multiple activities at once** and waits for all of them:

```
                 ┌─ ResizeActivity(12x12) ─┐
UploadWorkflow ──┤                         ├─→ CompleteActivity
                 └─ ResizeActivity(25x25) ─┘
```

"Fan-out" = launch N activities in parallel; "fan-in" = wait for all N. In Go you launch
each with `workflow.ExecuteActivity` (which returns a `Future` immediately, without
blocking) and then `.Get()` each future.

## 6. Two kinds of parallelism (workers)

- **Concurrency inside one worker:** a single worker process runs many activities at once on
  goroutines. So *one* worker already runs the 12×12 and 25×25 resizes simultaneously.
- **Multiple worker replicas:** run several identical worker containers polling the same task
  queue; Temporal load-balances activities across them (throughput + high availability). Our
  compose runs **2 replicas**, so you can watch the two resizes get picked up by *different*
  workers in the UI.

## 7. This project's workflow (walkthrough)

Files in `internal/pipeline/`:

- **`workflow.go` — `UploadWorkflow(ctx, UploadInput{ID, Ext, Filename})`**
  Note the input is a **reference** (an id + extension), **not the image bytes**. Temporal
  stores inputs in the event history and limits payload size, so large blobs stay on the
  shared `uploads` volume and activities read/write them there. The workflow: sets activity
  options → **fans out** a `Resize` activity per size in parallel → **fans in** (waits for
  all) → runs `Complete` (status → `complete`), or `Fail` (status → `failed`) if any resize
  ultimately fails.
- **`activities.go` — the real work.** `Resize` reads the original from storage, calls the
  gRPC image service, and saves the result. `Complete`/`Fail` update the Postgres status.
- **`orchestrator.go` — the Client side.** `Start(...)` calls `ExecuteWorkflow` and returns
  immediately; our HTTP server uses this so `POST /upload` responds `202 Accepted` right away.

End-to-end: `POST /upload` saves the original, records `processing`, starts the workflow, and
returns `202 + {id}`. Two worker replicas run the resizes in parallel; the workflow flips the
status to `complete`. The client polls `GET /images/{id}/status` until it reads `complete`,
then fetches the thumbnails. Watch the whole thing live at `http://localhost:8088`.

## 8. Testing (see `workflow_test.go`)

Temporal ships an in-memory **test environment** (`testsuite.WorkflowTestSuite`). Our tests
run the *real* `UploadWorkflow` with the activities **mocked** (`env.OnActivity(...)`), so we
verify the orchestration — fan-out happens, `Complete` runs on success, `Fail` runs on
failure — with no real gRPC, disk, or database.
