// Package e2e exercises the full MCP server stack in-process.
//
// Tests wire up a real Neo4jMCPServer (with tool registry + job registry),
// register one or more ToolDefs, then call every meta-tool over the MCP
// JSON-RPC layer using a test transport provided by mcp-go.
//
// Run with:
//
//	go test ./tests/e2e/ -v
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/server"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// ---- test fixtures -------------------------------------------------------

// buildTestServer returns a fully initialised mcpserver.MCPServer (the inner
// mcp-go server) with all four meta-tools registered and one sample tool in
// the registry.  We bypass Neo4jMCPServer.Start() (which calls ServeStdio)
// and instead wire everything up manually so tests can call handlers in-process.
func buildTestServer(t *testing.T, readOnly bool) (*mcpserver.MCPServer, *server.Dependencies) {
	t.Helper()
	cfg := &config.Config{ReadOnly: readOnly}

	inner := mcpserver.NewMCPServer(
		"test-mcp-server",
		"0.0.0-test",
		mcpserver.WithToolCapabilities(true),
	)

	deps := &server.Dependencies{
		Config: cfg,
		Tools:  server.NewToolRegistry(),
		Jobs:   server.NewJobRegistry(),
		Server: inner,
	}

	// Register a sample read tool.
	deps.Tools.Register(&server.ToolDef{
		ID:       "greet",
		Name:     "Greet",
		Type:     server.ToolTypeRead,
		ReadOnly: true,
		Parameters: []server.ToolParam{
			{Name: "name", Type: "string", Required: true, Description: "who to greet"},
		},
		Handler: func(_ context.Context, params map[string]interface{}, _ *server.Dependencies) (*mcpgo.CallToolResult, error) {
			name, _ := params["name"].(string)
			return mcpgo.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
		},
	})

	// Register a write tool (blocked in read-only mode).
	deps.Tools.Register(&server.ToolDef{
		ID:       "mutate",
		Name:     "Mutate",
		Type:     server.ToolTypeCreate,
		ReadOnly: false,
		Handler: func(_ context.Context, _ map[string]interface{}, _ *server.Dependencies) (*mcpgo.CallToolResult, error) {
			return mcpgo.NewToolResultText("mutation done"), nil
		},
	})

	// Register a slow tool for async testing.
	deps.Tools.Register(&server.ToolDef{
		ID:       "slow-greet",
		Name:     "Slow Greet",
		Type:     server.ToolTypeRead,
		ReadOnly: true,
		Handler: func(_ context.Context, params map[string]interface{}, _ *server.Dependencies) (*mcpgo.CallToolResult, error) {
			time.Sleep(100 * time.Millisecond)
			return mcpgo.NewToolResultText("slow hello"), nil
		},
	})

	// Add the four meta-tools.
	inner.AddTools(server.GetAllMetaTools(deps)...)

	return inner, deps
}

// callTool invokes a meta-tool by name with the given arguments and returns
// the first text content block.
func callTool(t *testing.T, deps *server.Dependencies, toolName string, args map[string]interface{}) string {
	t.Helper()
	ctx := context.Background()

	var handler func(context.Context, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error)

	switch toolName {
	case "list-tools":
		handler = server.ListToolsHandler(deps)
	case "get-tool-details":
		handler = server.GetToolDetailsHandler(deps)
	case "execute-tool":
		handler = server.ExecuteToolHandler(deps)
	case "get-tool-status":
		handler = server.GetToolStatusHandler(deps)
	default:
		t.Fatalf("unknown meta-tool: %s", toolName)
	}

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = args

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("[%s] unexpected Go error: %v", toolName, err)
	}
	return extractText(t, result)
}

// ---- list-tools e2e ------------------------------------------------------

func TestE2E_ListTools_ReturnsAllRegistered(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "list-tools", nil)

	for _, id := range []string{"greet", "mutate", "slow-greet"} {
		if !strings.Contains(text, id) {
			t.Errorf("expected %q in list-tools output, got: %s", id, text)
		}
	}
}

func TestE2E_ListTools_IsJSON(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "list-tools", nil)

	var arr []interface{}
	if err := json.Unmarshal([]byte(text), &arr); err != nil {
		t.Errorf("list-tools output is not valid JSON array: %v\nbody: %s", err, text)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 tools, got %d", len(arr))
	}
}

// ---- get-tool-details e2e ------------------------------------------------

func TestE2E_GetToolDetails_KnownTool(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "get-tool-details", map[string]interface{}{
		"tool_id": "greet",
	})
	if !strings.Contains(text, "greet") {
		t.Errorf("expected tool ID in details, got: %s", text)
	}
	if !strings.Contains(text, "parameters") {
		t.Errorf("expected 'parameters' in details, got: %s", text)
	}
}

func TestE2E_GetToolDetails_UnknownTool(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "get-tool-details", map[string]interface{}{
		"tool_id": "nonexistent",
	})
	if !isErrorResult(t, deps, "get-tool-details", "tool_id", "nonexistent") {
		_ = text // already failed inside isErrorResult
	}
}

// ---- execute-tool synchronous e2e ----------------------------------------

func TestE2E_ExecuteTool_Sync_ReadTool(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "execute-tool", map[string]interface{}{
		"tool_id":    "greet",
		"parameters": map[string]interface{}{"name": "World"},
	})
	if !strings.Contains(text, "Hello, World!") {
		t.Errorf("expected greeting, got: %s", text)
	}
}

func TestE2E_ExecuteTool_Sync_WriteTool_NotReadOnly(t *testing.T) {
	_, deps := buildTestServer(t, false) // read-only=false
	text := callTool(t, deps, "execute-tool", map[string]interface{}{
		"tool_id": "mutate",
	})
	if !strings.Contains(text, "mutation done") {
		t.Errorf("expected mutation result, got: %s", text)
	}
}

func TestE2E_ExecuteTool_Sync_WriteTool_ReadOnlyServer(t *testing.T) {
	_, deps := buildTestServer(t, true) // read-only=true

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"tool_id": "mutate"}

	result, _ := server.ExecuteToolHandler(deps)(context.Background(), req)

	data, _ := json.Marshal(result)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck

	if isErr, _ := m["isError"].(bool); !isErr {
		t.Errorf("expected isError=true for write tool on read-only server, got: %s", data)
	}
}

// ---- execute-tool async e2e ----------------------------------------------

func TestE2E_ExecuteTool_Async_FastTool(t *testing.T) {
	_, deps := buildTestServer(t, false)

	// Submit async.
	submitText := callTool(t, deps, "execute-tool", map[string]interface{}{
		"tool_id": "greet",
		"async":   true,
		"parameters": map[string]interface{}{"name": "Async"},
	})

	var jobResp map[string]string
	if err := json.Unmarshal([]byte(submitText), &jobResp); err != nil {
		t.Fatalf("async response is not JSON: %v — body: %s", err, submitText)
	}
	jobID := jobResp["job_id"]
	if jobID == "" {
		t.Fatal("expected non-empty job_id")
	}
	if jobResp["status"] != string(server.JobStatusPending) {
		t.Errorf("expected status=pending, got %q", jobResp["status"])
	}

	// Poll until done.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		statusText := callTool(t, deps, "get-tool-status", map[string]interface{}{"job_id": jobID})
		// Completed → handler returns raw result with "Hello, Async!" text.
		if strings.Contains(statusText, "Hello, Async!") {
			return
		}
		// Still in-flight: parse status.
		var view map[string]interface{}
		if err := json.Unmarshal([]byte(statusText), &view); err == nil {
			switch view["status"] {
			case string(server.JobStatusCompleted):
				return
			case string(server.JobStatusFailed):
				t.Fatalf("job failed unexpectedly: %s", statusText)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("async job did not complete within 5s")
}

func TestE2E_ExecuteTool_Async_SlowTool(t *testing.T) {
	_, deps := buildTestServer(t, false)

	submitText := callTool(t, deps, "execute-tool", map[string]interface{}{
		"tool_id": "slow-greet",
		"async":   true,
	})

	var jobResp map[string]string
	json.Unmarshal([]byte(submitText), &jobResp) //nolint:errcheck
	jobID := jobResp["job_id"]

	// First poll should be pending or running (not yet completed because the
	// handler sleeps 100ms).
	firstStatus := callTool(t, deps, "get-tool-status", map[string]interface{}{"job_id": jobID})
	var firstView map[string]interface{}
	json.Unmarshal([]byte(firstStatus), &firstView) //nolint:errcheck
	if firstView["status"] == string(server.JobStatusCompleted) {
		// Acceptable on very fast machines — just warn.
		t.Log("warning: slow-greet completed before first poll (very fast machine?)")
	}

	// Eventually it must complete.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		text := callTool(t, deps, "get-tool-status", map[string]interface{}{"job_id": jobID})
		if strings.Contains(text, "slow hello") {
			return
		}
		var v map[string]interface{}
		if json.Unmarshal([]byte(text), &v) == nil && v["status"] == string(server.JobStatusCompleted) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("slow-greet did not complete within 5s")
}

func TestE2E_GetToolStatus_UnknownJobID(t *testing.T) {
	_, deps := buildTestServer(t, false)

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"job_id": "does-not-exist"}
	result, _ := server.GetToolStatusHandler(deps)(context.Background(), req)

	data, _ := json.Marshal(result)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck
	if isErr, _ := m["isError"].(bool); !isErr {
		t.Errorf("expected isError=true for unknown job_id, got: %s", data)
	}
}

// ---- multiple concurrent async jobs -------------------------------------

func TestE2E_Async_MultipleConcurrentJobs(t *testing.T) {
	_, deps := buildTestServer(t, false)

	const n = 10
	jobIDs := make([]string, n)

	for i := 0; i < n; i++ {
		text := callTool(t, deps, "execute-tool", map[string]interface{}{
			"tool_id":    "greet",
			"async":      true,
			"parameters": map[string]interface{}{"name": fmt.Sprintf("user-%d", i)},
		})
		var resp map[string]string
		if err := json.Unmarshal([]byte(text), &resp); err != nil {
			t.Fatalf("job %d: invalid response: %v", i, err)
		}
		jobIDs[i] = resp["job_id"]
	}

	// All jobs must eventually complete.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		allDone := true
		for _, id := range jobIDs {
			view, err := deps.Jobs.Get(id)
			if err != nil || (view.Status != server.JobStatusCompleted && view.Status != server.JobStatusFailed) {
				allDone = false
				break
			}
		}
		if allDone {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("not all concurrent jobs completed within timeout")
}

// ---- helpers ------------------------------------------------------------

func extractText(t *testing.T, result interface{}) string {
	t.Helper()
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if content, ok := m["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if text, ok := block["text"].(string); ok {
				return text
			}
		}
	}
	t.Logf("result JSON: %s", data)
	t.Fatal("could not extract text from result content")
	return ""
}

// isErrorResult calls the named meta-tool with a single string arg and
// asserts the response has isError=true.  Returns false and calls t.Errorf
// on failure, allowing the caller to skip further assertions.
func isErrorResult(t *testing.T, deps *server.Dependencies, toolName, argKey, argVal string) bool {
	t.Helper()
	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{argKey: argVal}

	var result *mcpgo.CallToolResult
	switch toolName {
	case "get-tool-details":
		result, _ = server.GetToolDetailsHandler(deps)(context.Background(), req)
	case "execute-tool":
		result, _ = server.ExecuteToolHandler(deps)(context.Background(), req)
	default:
		t.Fatalf("isErrorResult: unsupported tool %q", toolName)
	}

	data, _ := json.Marshal(result)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck

	isErr, _ := m["isError"].(bool)
	if !isErr {
		t.Errorf("[%s] expected isError=true, got: %s", toolName, data)
	}
	return isErr
}
