# Upfluence Technical Assessment — Real-Time Post Analyzer

This service exposes an HTTP API (`/analysis`) that consumes a Server-Sent Events (SSE) stream from the Upfluence stream API, filters events, and calculates real-time aggregated metrics and percentiles over a given dimension and duration. The analysis results are returned synchronously once the aggregation window completes.

---

## Running the Service

### Prerequisites

* **Go 1.22+** (for local execution without Docker)
* **Docker & Docker Compose** (recommended)

### Configuration

The application can be configured using environment variables:

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port on which the HTTP server listens |
| `STREAM_URL` | `https://stream.upfluence.co/stream` | Upstream SSE stream endpoint to ingest |
| `EVENT_BUS_BUFFER_SIZE` | `100` | Size of the buffer for the event bus |

---

### Starting the Application

#### With Docker Compose (Recommended)

Build and start the service in a single command:

```bash
docker compose up --build
```

#### From GitHub Container Registry (Pre-built CI/CD Image)

Pull and run the pre-built image published automatically by the CI/CD pipeline:

```bash
docker run -p 8080:8080 ghcr.io/zipleix/upfluence-test:latest
```

#### Running Locally with Go

Download dependencies and run the server:

```bash
# Download dependencies
go mod download

# Start the server (or run 'make run')
go run main.go
```

#### Makefile Commands

A `Makefile` is provided to simplify standard development workflows:

```bash
make help        # Display all available targets
make run         # Run the service locally
make test-race   # Run tests with race detection (-race)
make lint        # Run golangci-lint
make lint-fix    # Automatically fix linting issues
```

---

## API Usage

### Endpoint: `GET /analysis`

The endpoint accepts two query parameters:
* `duration` *(optional)*: The time window for stream aggregation (e.g., `5s`, `10m`, `1h`). If omitted, the request remains open indefinitely until the client closes the connection.
* `dimension` *(required)*: The target metric to evaluate (`likes`, `comments`, `favorites`, `retweets`).

#### Bounded Request Example

```bash
curl "http://localhost:8080/analysis?duration=10s&dimension=likes"
```

#### Infinite / Unbounded Request Example

```bash
curl "http://localhost:8080/analysis?dimension=likes"
```

#### Response Example

```json
{
  "total_posts": 697,
  "minimum_timestamp": 1343635183,
  "maximum_timestamp": 1790164998,
  "likes_p50": 43,
  "likes_p90": 1895,
  "likes_p99": 87202
}
```

---

## CI/CD Pipeline & Container Registry

An automated CI/CD pipeline is configured with **GitHub Actions** ([.github/workflows/ci.yml](./.github/workflows/ci.yml)):

1. **Continuous Testing & Quality:** On every pull request and push to `master`, the pipeline executes the entire test suite with Go's race detector enabled (`go test -v -race ./...`) and computes statement coverage.
2. **Automated Docker Image Build:** If all tests succeed, a multi-stage, non-root, security-hardened Alpine container image is compiled.
3. **Publication to GHCR:** The built image is automatically pushed to the **GitHub Container Registry (GHCR)**:
   * **Registry URL:** `ghcr.io/zipleix/upfluence-test`
   * **Tagging Scheme:**
     * `:latest` on every push to `master`
     * `:v*.*.*` for Git release tags
     * `:<sha>` for specific commit traceability

To run the published image directly without local compilation:

```bash
docker pull ghcr.io/zipleix/upfluence-test:latest
docker run -p 8080:8080 ghcr.io/zipleix/upfluence-test:latest
```

---

## Technical Choices & Architecture

### Single Background Worker & In-Memory EventBus (Fan-out)

Instead of opening a new TCP/SSE connection to Upfluence for every incoming HTTP request, the service implements a **singleton stream worker** combined with an **in-memory fan-out EventBus**:

* **Upstream Protection & Quota Safety:** Opening multiple concurrent connections to an upstream SSE server leads to rate-limiting, IP bans, and unnecessary bandwidth consumption. The background worker maintains a single persistent connection.
* **Non-blocking Dispatch:** The EventBus distributes ingested items to all active client channels using non-blocking writes to guarantee that a slow consumer never blocks the main ingestion loop.
* **Automatic Reconnection:** If the upstream SSE connection drops, the worker automatically attempts reconnection with an exponential/linear backoff strategy without terminating the HTTP server.
* **Horizontal Scalability:** In an orchestrator like Kubernetes, each replica maintains a single upstream connection and serves its local concurrent requests efficiently.

### Polymorphic Domain Model (`Item` Interface)

The upstream SSE stream emits heterogeneous JSON payloads where field naming varies by platform (e.g., `favorites` on Twitter vs. `likes` on YouTube, or platform-specific engagement counters).

To decouple the transport and raw decoding layers from aggregation logic, the service uses an `Item` domain interface:

```go
type Item interface {
    Platform() string
    OccurredAt() time.Time
    Metric(dimension string) (int, bool)
}
```

* **Separation of Concerns:** The aggregation service does not need to know the specific platform schemas or manage cascade type assertions. It only interacts with the `Item` contract.
* **Extensibility Example (Adding `tiktok_video`):**
  Adding support for an unlisted platform requires zero refactoring of the core aggregation engine. It only takes two steps:

  1. Implement the `Item` interface on the model ([This this file for concret implementation](./service/aggregation/post.go)):
  ```go
  type TikTokVideo struct {
      ID        int64 `json:"id"`
      Likes     int   `json:"likes"`
      Comments  int   `json:"comments"`
      Timestamp int64 `json:"timestamp"`
  }

  func (t *TikTokVideo) Platform() string      { return "tiktok_video" }
  func (t *TikTokVideo) OccurredAt() int64 { return t.Timestamp }
  func (t *TikTokVideo) Metric(dim string) (int, bool) {
      switch dim {
      case "likes":
          return t.Likes, true
      case "comments":
          return t.Comments, true
      default:
          return 0, false
      }
  }
  ```

  2. Bind the new type in `StreamEvent.AsItem()`:
  ```go
  case e.TikTokVideo != nil:
      return e.TikTokVideo, true
  ```

---

### Contextual Observability & Structured Logging

The service implements structured logging using Go's standard library `log/slog`, organized around correlation IDs and thread-safe request context propagation:

* **Correlation via `X-Request-ID`:** Every incoming request receives a unique correlation ID (preserved from incoming headers or generated as a random hex string). It is injected into the context and propagated back in the response headers.
* **Dynamic Context Accumulator (`logger.Add`):** The custom `logger` package injects a mutex-protected attribute container into the request `context.Context`. Handlers can enrich the request context with metadata at any point during execution (e.g., `requested_duration`, `dimension`, or internal errors) without polluting function signatures.
* **Single Completion Entry:** The HTTP logging middleware intercepts the request lifecycle. Upon completion, it extracts all dynamic attributes accumulated in the context and emits a single, structured log entry containing the HTTP method, path, query parameters, status code, execution duration, client IP, and contextual attributes.
* **Context-Aware Logger:** The `logger.FromContext(ctx)` helper allows any layer of the application to retrieve a pre-configured `*slog.Logger` instance enriched with all request-scoped attributes collected up to that point.

## Trade-offs & Engineering Decisions

### 1. Percentile Calculation: Exact In-Memory vs. Constant-Memory Sketching

* **Current Implementation:** Incoming dimension values are accumulated in an in-memory slice (`[]int`) and sorted using `sort.Ints` when the aggregation window closes. Percentiles (p50, p90, p99) are extracted directly via nearest-rank indexing.
* **Trade-off:** This provides 100% mathematical precision at the expense of memory consumption ($O(N)$ space complexity) and final sorting latency ($O(N \log N)$ CPU burst).
* **Production Alternative:** For unbounded streams running over days or weeks with millions of events, retaining every integer in memory introduces an Out-Of-Memory (OOM) risk. The production-grade approach is to replace the slice with a streaming quantile approximation data structure (such as **T-Digest**, **HdrHistogram**, or **Greenwald-Khanna**). These algorithms guarantee strict $O(1)$ constant memory usage while retaining bounded relative error.

---

### 2. Dimension Filtering & Total Post Accounting

* **Current Implementation:** Posts that do not natively support the requested dimension are filtered out before being added to `total_posts` or the timestamp boundary tracking.
* **Rationale:** If a client requests `retweets` (a metric exclusive to Twitter), including YouTube videos or blog articles with a default value of 0 would heavily distort percentiles (artificially forcing the median to 0) and misrepresent the actual population analyzed. Therefore, `total_posts`, `minimum_timestamp`, and `maximum_timestamp` strictly describe the sample of posts eligible for the requested dimension.

---

### 3. Non-Documented Event Types (`tiktok_video`, `twitch_stream`, `story`)

* **Current Implementation:** The upstream stream emits event types not specified in the initial requirements (e.g., `tiktok_video`, `twitch_stream`, `story`). By default, these events are safely ignored during extraction.
* **Rationale:** Strictly respecting the initial contract prevents unexpected metric collisions and ensures deterministic statistical results. However, thanks to the polymorphic `Item` design, enabling any of these types requires only a few lines in the model without touching the aggregation pipeline.

---

### 4. Synchronous Long-Polling vs. Asynchronous Job Processing

* **Current Implementation:** The API keeps the client HTTP request open for the entire duration specified by `duration` (with `WriteTimeout: 0`) and returns the final JSON response synchronously.
* **Production Consideration:** Holding HTTP connections open for long intervals (e.g., several hours) is vulnerable to network dropouts, reverse-proxy timeouts (e.g., NGINX/Cloudflare 60s defaults), and memory pressure. This pattern should be refactored into an asynchronous job workflow (`POST /analysis` returning a `202 Accepted` with a job ID, computing in the background, and serving results via polling, webhooks or HTTP request).

---

## What to Do with More Time (Future Improvements)

While the current service is modular, tested, and production-ready for the scope of the assessment, the following enhancements would be prioritized given additional time:

### 1. Constant-Memory Quantile Sketches (T-Digest / HdrHistogram)
* **Goal:** Eliminate the $O(N)$ memory growth and $O(N \log N)$ sorting spike at request completion.
* **Approach:** Replace the raw integer slice accumulator with an approximation sketch algorithm such as **T-Digest** or **HdrHistogram**. This would bound memory allocation to a deterministic constant size (a few kilobytes per concurrent query) regardless of stream throughput or analysis duration.

### 2. Backpressure & Congestion Metrics on the EventBus
* **Goal:** Increase visibility when the system operates under heavy traffic.
* **Approach:** The in-memory fan-out channel currently drops events for lagging client buffers to prevent blocking the worker. Adding drop counters exposed via Prometheus metrics (e.g., `eventbus_dropped_events_total`) or implementing adaptive subscriber throttling would allow detecting slow consumers and preventing silent data bias.

### 3. Persistent Storage & Event Replay Engine
* **Goal:** Support historical querying and resilience across container restarts.
* **Approach:** Currently, an analysis request only collects events that arrive *after* the request is issued. Decoupling ingestion from querying by storing events in an append-only time-series or columnar database (such as **ClickHouse** or **TimescaleDB**) would enable instant queries over past time ranges (e.g., *"give me metrics for the last 2 hours"*).

### 4. Asynchronous Job Workflow (Long-Running Tasks)
* **Goal:** Protect network infrastructure from held-open HTTP connections.
* **Approach:** Replace synchronous long-polling with a standard job orchestration model:
  1. `POST /analysis` creates an aggregation task and immediately returns `202 Accepted` with a `job_id`.
  2. A background worker executes the time-bounded window.
  3. The client retrieves the result via `GET /analysis/{job_id}` (polling) or receives a webhook notification upon completion.

### 5. Production Health & Telemetry Probes
* **Goal:** Enable proper Kubernetes integration and operational alerting.
* **Approach:** Implement `/healthz` (liveness: is the server running?) and `/ready` (readiness: is the upstream SSE worker actively connected and receiving data?). Export OpenTelemetry metrics for active subscribers, SSE latency, upstream reconnect counts, and HTTP request distributions.
