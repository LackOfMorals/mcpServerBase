# MCP Server Base

A foundational [Model Context Protocol](https://modelcontextprotocol.io) server written in Go. Clone it, add your own tools, and connect any MCP-compatible client.

---

## Features

- **Stdio and HTTP/HTTPS transports** — switch with a single flag
- **API-key authentication** for HTTP mode, with a configurable bypass list for public MCP methods
- **Async tool execution** — run long-running tools in the background and poll for results
- **Progress notifications** — stream status updates to the client while a tool runs
- **Four meta-tools** always available: `list-tools`, `get-tool-details`, `execute-tool`, `get-tool-status`
- **Read-only mode** — disables all write tools at the server level

---

## Prerequisites

- Go 1.25+ (see `go.mod`)

---

## Installation

```bash
git clone https://github.com/LackOfMorals/mcpServerBase
cd mcpServerBase
go mod download
```

---

## Build

```bash
go build -o ./bin/mcp-base ./cmd/mcp-base
```

---

## Running

### stdio mode (default)

```bash
./bin/mcp-base
```

### HTTP mode

```bash
./bin/mcp-base --transport=http --api-key=mysecret
```

The MCP endpoint is available at `http://127.0.0.1:6666/mcp`.

### HTTPS mode

```bash
./bin/mcp-base \
  --transport=http \
  --tls \
  --tls-cert=/etc/certs/server.crt \
  --tls-key=/etc/certs/server.key \
  --api-key=mysecret
```

---

## Configuration

Every setting can be supplied as a CLI flag or an environment variable. CLI flags take precedence.

| Setting | Flag | Env var | Default |
|---|---|---|---|
| Read-only mode | `--read-only` | `READ_ONLY` | `true` |
| Log level | `--log-level` | `LOG_LEVEL` | `info` |
| Log format | `--log-format` | `LOG_FORMAT` | `text` |
| Transport | `--transport` | `TRANSPORT` | `stdio` |
| HTTP listen address | `--http-addr` | `HTTP_ADDR` | `127.0.0.1:6666` |
| Enable TLS | `--tls` | `TLS` | `false` |
| TLS certificate | `--tls-cert` | `TLS_CERT_FILE` | — |
| TLS private key | `--tls-key` | `TLS_KEY_FILE` | — |
| API key | `--api-key` | `API_KEY` | — |

When using HTTP mode, the following MCP methods are accessible without authentication by default: `initialize`, `notifications/initialized`, `ping`, `tools/list`.

---

## Adding Tools

Tools live in `internal/tools/`. Adding one requires two things: a handler function and a catalog entry.

### Step 1 — Write the handler

Create a new file in `internal/tools/` for your tool. The handler must match the `ToolHandler` signature:

```go
func(ctx context.Context, parameters map[string]interface{}, deps *tools.Dependencies) (*mcp.CallToolResult, error)
```

The `hello-world` tool (`internal/tools/hello.go`) is the simplest possible example:

```go
package tools

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
)

func helloWorldHandler(_ context.Context, _ map[string]interface{}, _ *Dependencies) (*mcp.CallToolResult, error) {
    return mcp.NewToolResultText("Hello, World!"), nil
}
```

For a tool that accepts parameters, read them from the `parameters` map:

```go
func greetHandler(_ context.Context, parameters map[string]interface{}, _ *Dependencies) (*mcp.CallToolResult, error) {
    name, _ := parameters["name"].(string)
    if name == "" {
        name = "World"
    }
    return mcp.NewToolResultText("Hello, " + name + "!"), nil
}
```

Return an MCP-level error (visible to the LLM) using `mcp.NewToolResultError`:

```go
func safeHandler(_ context.Context, parameters map[string]interface{}, _ *Dependencies) (*mcp.CallToolResult, error) {
    id, ok := parameters["id"].(string)
    if !ok || id == "" {
        return mcp.NewToolResultError("id parameter is required"), nil
    }
    // ... do work ...
    return mcp.NewToolResultText("done"), nil
}
```

Use a Go `error` return for unexpected failures (network errors, panics, etc.) — the server converts these to MCP error results automatically.

### Step 2 — Register in the catalog

Open `internal/tools/catalog.go` and add a `ToolDef` entry to the slice returned by `catalog()`:

```go
func catalog() []*ToolDef {
    return []*ToolDef{
        {
            ID:          "hello-world",
            Name:        "Hello World",
            Description: "Returns 'Hello, World!' — a minimal example tool.",
            Type:        ToolTypeRead,
            ReadOnly:    true,
            Handler:     helloWorldHandler,
        },
        {
            ID:          "greet",
            Name:        "Greet",
            Description: "Returns a personalised greeting.",
            Type:        ToolTypeRead,
            ReadOnly:    true,
            Parameters: []ToolParam{
                {Name: "name", Type: "string", Description: "Name to greet", Required: false},
            },
            Handler: greetHandler,
        },
    }
}
```

That's it. Rebuild and the new tool appears automatically in `list-tools`.

### ToolDef fields

| Field | Type | Description |
|---|---|---|
| `ID` | `string` | Unique identifier used by `execute-tool` |
| `Name` | `string` | Human-readable display name |
| `Description` | `string` | Shown to the LLM in `get-tool-details` |
| `Type` | `ToolType` | One of `list`, `read`, `create`, `update`, `delete` |
| `ReadOnly` | `bool` | When `true`, the tool is allowed even in read-only mode |
| `Parameters` | `[]ToolParam` | Declared inputs (optional) |
| `Handler` | `ToolHandler` | The function that executes the tool |

### ToolType values

| Constant | Value | Use when the tool… |
|---|---|---|
| `ToolTypeList` | `"list"` | Returns a collection of items |
| `ToolTypeRead` | `"read"` | Reads data without side effects |
| `ToolTypeCreate` | `"create"` | Creates a new resource |
| `ToolTypeUpdate` | `"update"` | Modifies an existing resource |
| `ToolTypeDelete` | `"delete"` | Removes a resource |

### ToolParam fields

| Field | Type | Description |
|---|---|---|
| `Name` | `string` | Parameter name as the caller supplies it |
| `Type` | `string` | JSON type: `string`, `integer`, `number`, `boolean`, `object`, `array` |
| `Description` | `string` | Shown to the LLM |
| `Required` | `bool` | Whether the parameter must be present |
| `Default` | `interface{}` | Optional default value |

### Using Dependencies in a handler

The `deps *Dependencies` argument gives access to server-level state:

```go
type Dependencies struct {
    Config *config.Config        // server configuration (e.g. ReadOnly flag)
    Tools  *ToolRegistry         // registered tools
    Jobs   *JobRegistry          // async job store
    Server *mcpgoserver.MCPServer // for sending progress notifications
}
```

### Sending progress notifications

For long-running tools, use `tools.NewProgressSender` to stream status updates to the client:

```go
func longRunningHandler(ctx context.Context, parameters map[string]interface{}, deps *Dependencies) (*mcp.CallToolResult, error) {
    req := mcp.CallToolRequest{} // populated by the framework
    ps := tools.NewProgressSender(ctx, deps.Server, req)

    ps.Step(1, 3, "Starting...")
    // ... do work ...
    ps.Step(2, 3, "Halfway there...")
    // ... do more work ...
    ps.Step(3, 3, "Done")

    return mcp.NewToolResultText("completed"), nil
}
```

`ProgressSender` is a no-op when the client did not supply a progress token, so it is always safe to call.

### Async execution

Any tool can be run asynchronously without code changes. The caller passes `"async": true` in the `execute-tool` call and polls `get-tool-status` with the returned `job_id` until the status is `completed` or `failed`.

---

## Testing

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# Specific packages
go test ./tests/unit/...
go test ./tests/e2e/...
go test ./tests/transport/...
go test ./internal/transport/http/...
```

See `tests/README.md` for full details.

---

## Testing with MCP Inspector

```bash
npx @modelcontextprotocol/inspector go run ./cmd/mcp-base
```

---

## Using with Claude Desktop

Add the following to your Claude Desktop configuration file:

**stdio mode**

```json
{
  "mcpServers": {
    "mcp-base": {
      "command": "/full/path/to/bin/mcp-base",
      "args": [],
      "env": {
        "READ_ONLY": "false"
      }
    }
  }
}
```

**HTTP mode** — start the server separately, then configure Claude Desktop to connect to it:

```json
{
  "mcpServers": {
    "mcp-base": {
      "command": "/full/path/to/bin/mcp-base",
      "args": ["--transport=http", "--api-key=mysecret"],
      "env": {}
    }
  }
}
```

---

## Project Layout

```
cmd/mcp-base/       main entry point
internal/
  cli/              CLI flag parsing
  config/           configuration loading and validation
  logger/           slog wrapper
  server/           MCP server lifecycle (transport selection)
  tools/            tool types, registry, async engine, meta-tools
    catalog.go      register your tools here
    hello.go        example tool
  transport/
    stdio/          stdio transport
    http/           HTTP/HTTPS transport with auth middleware
tests/
  unit/             isolated unit tests
  e2e/              full in-process stack tests
  transport/        HTTP transport integration tests
```
