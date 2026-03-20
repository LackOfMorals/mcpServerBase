# Tests

Tests are split into two packages under this directory:

```
tests/
├── unit/   – fast, isolated tests (no I/O, no goroutine sleeps > 50 ms)
└── e2e/    – end-to-end tests that wire a full server and exercise all
               four meta-tools in-process
```

## Running

```bash
# All tests
go test ./tests/...

# Unit only
go test ./tests/unit/... -v

# E2E only
go test ./tests/e2e/... -v

# With race detector (recommended in CI)
go test -race ./tests/...
```

## Unit tests (`tests/unit/`)

| File | What it covers |
|---|---|
| `helpers_test.go` | Shared fixtures — `newDeps`, sample handlers, request builders |
| `tool_registry_test.go` | `ToolRegistry`: `Register` (panic on duplicate), `GetAllSummaries`, `GetTool`, `ExecuteTool` (success, unknown, read-only block, nil handler), `AsyncExecuteTool` |
| `async_test.go` | `JobRegistry`: status lifecycle (pending → running → completed / failed), context cancellation, not-found error, concurrent-submit safety, result content population |
| `tool_handlers_test.go` | All four meta-tool handlers: `ListToolsHandler`, `GetToolDetailsHandler`, `ExecuteToolHandler` (sync + async), `GetToolStatusHandler` |
| `tool_types_test.go` | JSON serialisation of `ToolDef` (handler excluded), `ToolSummary`, `ToolParam` (default omitted/included); constant values for `ToolType` and `JobStatus` |

## E2E tests (`tests/e2e/`)

| Test | Scenario |
|---|---|
| `TestE2E_ListTools_*` | Returns all registered tools as a valid JSON array |
| `TestE2E_GetToolDetails_*` | Known tool includes parameters; unknown tool returns `isError=true` |
| `TestE2E_ExecuteTool_Sync_*` | Read tool succeeds; write tool succeeds / blocked when server is read-only |
| `TestE2E_ExecuteTool_Async_FastTool` | Returns `job_id` immediately; polling resolves to tool output |
| `TestE2E_ExecuteTool_Async_SlowTool` | First poll returns pending/running; eventually completes |
| `TestE2E_GetToolStatus_UnknownJobID` | Returns `isError=true` |
| `TestE2E_Async_MultipleConcurrentJobs` | 10 concurrent async submissions all complete |

## Adding new tools

Register a `*server.ToolDef` in `main.go` via `mcpServer.RegisterTool(...)` before calling `Start()`.
The four meta-tools (`list-tools`, `get-tool-details`, `execute-tool`, `get-tool-status`) are
registered automatically and require no changes.
