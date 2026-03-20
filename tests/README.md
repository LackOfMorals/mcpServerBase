# Tests

Tests are organised in three packages under this directory, plus one set of
tests that live alongside the transport source code:

```
tests/
├── unit/       – fast, isolated tests (no I/O, no goroutine sleeps >50 ms)
├── transport/  – integration tests for the HTTP transport layer
├── e2e/        – full in-process MCP stack tests
└── README.md   – this file

internal/transport/http/
├── middleware_test.go  – white-box unit tests for auth + logging middleware
```

## Running

```bash
# Everything
go test ./...

# Unit tests only
go test ./tests/unit/... -v

# HTTP transport integration tests
go test ./tests/transport/... -v

# E2E tests only
go test ./tests/e2e/... -v

# Middleware white-box tests (alongside source)
go test ./internal/transport/http/... -v

# With race detector (recommended in CI)
go test -race ./...
```

---

## Unit tests (`tests/unit/`)

| File | What it covers |
|---|---|
| `helpers_test.go` | Shared fixtures — `newDeps`, sample handlers, request builders |
| `tool_registry_test.go` | `ToolRegistry`: Register, GetAllSummaries, GetTool, ExecuteTool, AsyncExecuteTool |
| `async_test.go` | `JobRegistry`: status lifecycle, context cancellation, concurrency, GetResult |
| `tool_handlers_test.go` | All four meta-tool handlers (list, details, execute sync/async, status) |
| `tool_types_test.go` | JSON serialisation of ToolDef/ToolSummary/ToolParam; ToolType/JobStatus constants |
| `config_test.go` | Config loading: transport selection, TLS validation, API key, CLI-over-env precedence |

---

## HTTP transport integration tests (`tests/transport/`)

These use `httptest.NewServer` and `httpsvr.NewHandler` to spin up a real
in-process HTTP server with the full middleware chain wired up.  No MCP
protocol is exercised — a simple echo backend is used so tests focus
entirely on the HTTP layer.

| Test | Scenario |
|---|---|
| `TestHTTP_NoAPIKey_AllRequestsPass` | When no API key is configured every method passes |
| `TestHTTP_PublicMethods_NoAuthRequired` | Default public methods (initialize, ping, tools/list, notifications/initialized) bypass auth |
| `TestHTTP_ProtectedMethod_NoKey_Returns401` | Protected method without a key → 401 |
| `TestHTTP_ProtectedMethod_WrongKey_Returns401` | Wrong Bearer token → 401 |
| `TestHTTP_ProtectedMethod_BearerToken_Returns200` | Correct `Authorization: Bearer` → 200 |
| `TestHTTP_ProtectedMethod_XAPIKey_Returns200` | Correct `X-API-Key` header → 200 |
| `TestHTTP_GET_WithValidKey_Passes` | Authenticated GET (SSE stream initiation) → 200 |
| `TestHTTP_GET_NoKey_Returns401` | Unauthenticated GET → 401 |
| `TestHTTP_BodyForwardedToBackend` | Middleware peek does not consume the request body |
| `TestHTTP_CustomPublicMethods_OnlyListedBypass` | Custom public method list is respected |

---

## Middleware white-box tests (`internal/transport/http/middleware_test.go`)

In-package tests (`package httpsvr`) that directly exercise unexported helpers.

| Test group | Coverage |
|---|---|
| `TestCheckAPIKey_*` | Bearer match, X-API-Key match, wrong/missing/empty tokens, non-Bearer scheme |
| `TestBuildPublicMethodSet_*` | Set membership, non-membership, empty input |
| `TestAPIKeyAuth_*` | No key (open), public bypass, valid Bearer/X-API-Key, wrong key, GET auth, body restored after peek |
| `TestLoggingMiddleware_*` | Status codes captured, downstream called, default 200 |
| `TestResponseWriter_*` | WriteHeader called only once |

---

## E2E tests (`tests/e2e/`)

Full in-process stack tests — a real `MCPServer` + tool registry + job registry
is wired and every meta-tool exercised through the handler layer.

| Test | Scenario |
|---|---|
| `TestE2E_ListTools_*` | Returns all registered tools as a valid JSON array |
| `TestE2E_GetToolDetails_*` | Known tool returns parameters; unknown → isError=true |
| `TestE2E_ExecuteTool_Sync_*` | Read tool succeeds; write blocked in read-only mode |
| `TestE2E_ExecuteTool_Async_*` | Returns job_id; fast and slow tools eventually complete |
| `TestE2E_GetToolStatus_*` | Unknown job_id → isError=true |
| `TestE2E_Async_MultipleConcurrentJobs` | 10 concurrent async jobs all complete |

---

## Adding new tools

Register a `*tools.ToolDef` in `cmd/mcp-base/main.go` via `mcpServer.RegisterTool(...)` before calling `Start()`. The four meta-tools and both transport modes are automatically available with no additional changes.

## Transport configuration quick reference

| Setting | CLI flag | Env var | Default |
|---|---|---|---|
| Transport mode | `--transport` | `TRANSPORT` | `stdio` |
| HTTP listen address | `--http-addr` | `HTTP_ADDR` | `127.0.0.1:6666` |
| Enable TLS | `--tls` | `TLS` | `false` |
| TLS certificate | `--tls-cert` | `TLS_CERT_FILE` | — |
| TLS private key | `--tls-key` | `TLS_KEY_FILE` | — |
| API key | `--api-key` | `API_KEY` | — |
