# Search infrastructure and loading skeletons — Design

**Date:** 2026-11-15
**Status:** Approved

## Goal

Add reusable search infrastructure to generated Go/Gin backends. Applications can
index documents through a provider-neutral Go API and expose a public search
endpoint backed by Meilisearch.

Also add an explicit loading-skeleton rule to the generated Next.js app's
`CONTEXT.md`.

## Scope

This slice adds:

- `search:meilisearch` for all four Go/Gin stacks.
- A provider-neutral `internal/search` core.
- A Meilisearch HTTP adapter using the standard library.
- A public `GET /api/v1/search` endpoint.
- Meilisearch Docker Compose files, configuration, docs, and tests.
- A loading-skeleton rule in `templates/frontend/next/files/CONTEXT.md`.

This slice does not add:

- A search UI.
- A document CRUD API.
- Authentication on the public query endpoint.
- Elasticsearch.
- Queue-backed indexing in the core slice; it ships as the separate `search-sync:rabbitmq` follow-up.

The queue-backed `search-sync:rabbitmq` overlay ships as the implemented follow-up.

## Decisions

- Meilisearch is the first provider because it starts as one container, has a
  small operational surface, and provides typo tolerance without cluster
  management.
- The core stays provider-neutral so Elasticsearch can implement the same
  interfaces later.
- The public endpoint is read-only. Indexing is an internal Go API, not an HTTP
  route.
- The endpoint searches one configured index. The index must contain public
  documents because the endpoint is unauthenticated.
- No new Go module dependency is required; the adapter uses `net/http`.
- Search uses the locally available `getmeili/meilisearch:v1.6` image.

## Architecture

### Go packages

```text
internal/search/
  models.go        Document, Options, Result, errors
  ports.go         Indexer and Searcher interfaces
  service.go       validation and provider-neutral orchestration
  http/
    handler.go     public GET /api/v1/search
    api_types.go   HTTP response types
    spec.go        embedded OpenAPI document
    openapi.yaml   search contract
    handler_test.go
  meilisearch/
    client.go      stdlib HTTP adapter
    client_test.go
internal/config/
  config_search.go
  config_search_test.go
compose.search.yaml
compose.override.search.yaml
docs/SEARCH.md
```

The core interfaces are:

```go
type Document struct {
	ID     string
	Fields map[string]any
}

type Options struct {
	Limit  int
	Offset int
}

type Result struct {
	Query              string
	Hits               []json.RawMessage
	EstimatedTotalHits int
	Limit              int
	Offset             int
	ProcessingTimeMs   int
}

type Indexer interface {
	Upsert(ctx context.Context, documents []Document) error
	Delete(ctx context.Context, ids []string) error
}

type Searcher interface {
	Search(ctx context.Context, query string, options Options) (Result, error)
}
```

`Service` validates query length and pagination, then delegates to `Searcher`.
`Upsert` and `Delete` delegate to `Indexer`; they are not exposed by HTTP.

### Meilisearch adapter

The adapter stores the base URL, index UID, API key, timeout, and an
`*http.Client`.

Meilisearch calls:

- `POST /indexes/{index}/search` with `q`, `limit`, and `offset`.
- `POST /indexes/{index}/documents?primaryKey=id` with an array of documents.
- `POST /indexes/{index}/documents/delete-batch` with document IDs.

The first upsert creates the index with `id` as the primary key. Upserts and
deletes are synchronous from the caller's perspective. Meilisearch task
completion is asynchronous; the infrastructure documents that indexing calls
return after Meilisearch accepts the task, not after the task finishes.

If the configured index does not exist yet, the search adapter returns an empty
result with the requested page bounds instead of treating a first-run search as
an outage.

`Search` sends `Authorization: Bearer <SEARCH_API_KEY>` when the key is set.
Non-2xx responses are classified as unavailable or invalid and wrapped with
context. API keys and document bodies never appear in errors or logs.

### HTTP contract

`GET /api/v1/search?q=<query>&limit=20&offset=0`

- `q`: required, trimmed, 1–200 characters.
- `limit`: optional, default 20, 1–`SEARCH_MAX_LIMIT`.
- `offset`: optional, default 0, 0–1000.

Success:

```json
{
  "query": "design systems",
  "hits": [{ "id": "doc-1", "title": "Design systems" }],
  "estimated_total_hits": 1,
  "limit": 20,
  "offset": 0,
  "processing_time_ms": 3
}
```

Errors use the existing RFC 9457 shape:

- `422 validation_failed` for invalid query or pagination.
- `503 search_unavailable` when Meilisearch cannot serve the request.
- `500 internal_error` for unexpected failures.

The route is public and uses the app's existing rate limiting, timeout, CORS,
request-ID, and logging middleware. The OpenAPI document is merged into Swagger
UI like other feature documents.

### Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `SEARCH_PORT` | `7700` | Docker host port |
| `SEARCH_URL` | `http://localhost:7700` | API-to-Meilisearch URL |
| `SEARCH_INDEX` | `documents` | Search index UID |
| `SEARCH_API_KEY` | generated secret | Meilisearch master/API key |
| `SEARCH_TIMEOUT` | `5s` | Per-request upstream timeout |
| `SEARCH_MAX_LIMIT` | `50` | Maximum page size |

Validation requires an HTTP(S) URL, a non-empty index matching
`[A-Za-z0-9_-]+`, a positive timeout, and a max limit between 1 and 100.

### Docker Compose

`compose.search.yaml` starts Meilisearch for local development.
`compose.override.search.yaml` adds Meilisearch to the full stack and points the
API at `http://search:7700`.

```yaml
search:
  image: getmeili/meilisearch:v1.6
  environment:
    MEILI_ENV: development
    MEILI_MASTER_KEY: ${SEARCH_API_KEY:-smokeforge-dev-search-key}
    MEILI_NO_ANALYTICS: "true"
  ports:
    - "${SEARCH_PORT:-7700}:7700"
  volumes:
    - search-data:/meili_data
```

Make targets: `search-up`, `search-down`, `search-logs`, `search-shell`.

The API starts independently of Meilisearch. Search requests return 503 until
the engine is reachable. Readiness does not include search in this slice.

### Indexing usage

Application code injects `search.Indexer` where its domain data changes:

```go
err := indexer.Upsert(ctx, []search.Document{{
	ID: "doc-1",
	Fields: map[string]any{"title": "Design systems"},
}})
```

There is no seed command and no demo document. Applications own their index
schema and document IDs.

## Security and data rules

- The query endpoint is public. Only public documents belong in the configured
  index.
- Indexing is not exposed over HTTP.
- `SEARCH_API_KEY` is server-only and never logged.
- Document fields are returned as opaque JSON; the API does not reshape or
  trust them.
- Query text and pagination are validated before reaching Meilisearch.
- No user-controlled URLs are fetched by the API; the Meilisearch URL is
  configuration only.

## Error handling and tests

- Wrap errors with lowercase context.
- Use sentinel errors for invalid input and unavailable search.
- Handler tests cover success, invalid query, and upstream failure.
- Adapter tests use `httptest.Server` to verify paths, request bodies,
  authorization, response parsing, and non-2xx mapping.
- Service tests cover query trimming, length limits, pagination defaults, and
  bounds.
- Config tests cover URL, index UID, timeout, and limit validation.
- The full four-stack matrix generates `search:meilisearch` and checks tests,
  vet, build, docs, and Compose configuration.

## Frontend loading skeletons

Add this rule to `templates/frontend/next/files/CONTEXT.md` under the responsive
and performance floor:

> Use loading skeletons for asynchronous content and route-level loading states.
> Match the final layout and dimensions, keep them accessible with `aria-busy` or
> a live status where appropriate, and reserve spinners for short inline actions
> rather than content regions.

This is guidance for future UI work; no search UI is added in this slice.

## Queue-backed indexing follow-up

`search-sync:rabbitmq` requires `search:meilisearch` and `queue:rabbitmq`. It
implements `search.Indexer` by publishing `search.documents.upsert` and
`search.documents.delete` jobs, and registers worker handlers that call the
Meilisearch adapter. The API now creates a RabbitMQ connection manager and
publisher for this overlay, declares the existing topology, and closes both on
shutdown. Query behavior is unchanged; indexing becomes eventually consistent.

## Non-goals

- Search relevance tuning, synonyms, ranking rules, or typo configuration.
- Multi-index or tenant-scoped search.
- Search UI or command palette.
- Elasticsearch-specific query features.
- Automatic backfill from an existing database.
