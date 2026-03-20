// Package e2e exercises the full MCP server stack in-process.
//
// Tests wire up a real MCPServer (with tool registry + job registry),
// register sample ToolDefs, then call every meta-tool through the actual
// handler layer so the full tools package is exercised end-to-end.
//
// Run with:
//
//	go test ./tests/e2e/ -v
//	go test -race ./tests/e2e/ -v
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/tools"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// ---- test fixtures -------------------------------------------------------

// buildTestServer returns a wired *tools.Dependencies and the underlying
// *mcpgoserver.MCPServer with all four meta-tools already registered.
// We bypass Neo4jMCPServer.Start() (which calls ServeStdio) and instead
// wire everything manually so handlers can be called in-process.
func buildTestServer(t *testing.T, readOnly bool) (*mcpgoserver.MCPServer, *tools.Dependencies) {
	t.Helper()

	inner := mcpgoserver.NewMCPServer(
		"test-mcp-server",
		"0.0.0-test",
		mcpgoserver.WithToolCapabilities(true),
	)

	deps := &tools.Dependencies{
		Config: &config.Config{ReadOnly: readOnly},
		Tools:  tools.NewToolRegistry(),
		Jobs:   tools.NewJobRegistry(),
		Server: inner,
	}

	// A simple read tool.
	deps.Tools.Register(&tools.ToolDef{
		ID:       "greet",
		Name:     "Greet",
		Type:     tools.ToolTypeRead,
		ReadOnly: true,
		Parameters: []tools.ToolParam{
			{Name: "name", Type: "string", Required: true, Description: "who to greet"},
		},
		Handler: func(_ context.Context, params map[string]interface{}, _ *tools.Dependencies) (*mcpgo.CallToolResult, error) {
			name, _ := params["name"].(string)
			return mcpgo.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
		},
	})

	// A write tool — blocked when the server is read-only.
	deps.Tools.Register(&tools.ToolDef{
		ID:       "mutate",
		Name:     "Mutate",
		Type:     tools.ToolTypeCreate,
		ReadOnly: false,
		Handler: func(_ context.Context, _ map[string]interface{}, _ *tools.Dependencies) (*mcpgo.CallToolResult, error) {
			return mcpgo.NewToolResultText("mutation done"), nil
		},
	})

	// A slow tool for async testing.
	deps.Tools.Register(&tools.ToolDef{
		ID:       "slow-greet",
		Name:     "Slow Greet",
		Type:     tools.ToolTypeRead,
		ReadOnly: true,
		Handler: func(_ context.Context, _ map[string]interface{}, _ *tools.Dependencies) (*mcpgo.CallToolResult, error) {
			time.Sleep(100 * time.Millisecond)
			return mcpgo.NewToolResultText("slow hello"), nil
		},
	})

	inner.AddTools(tools.GetAllMetaTools(deps)...)

	return inner, deps
}

// callTool invokes a meta-tool handler by name and returns the first text
// content block from the result.
func callTool(t *testing.T, deps *tools.Dependencies, toolName string, args map[string]interface{}) string {
	t.Helper()

	var handler func(context.Context, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error)
	switch toolName {
	case "list-tools":
		handler = tools.ListToolsHandler(deps)
	case "get-tool-details":
		handler = tools.GetToolDetailsHandler(deps)
	case "execute-tool":
		handler = tools.ExecuteToolHandler(deps)
	case "get-tool-status":
		handler = tools.GetToolStatusHandler(deps)
	default:
		t.Fatalf("unknown meta-tool: %s", toolName)
	}

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = args

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("[%s] unexpected Go error: %v", toolName, err)
	}
	return extractText(t, result)
}

// ---- list-tools ----------------------------------------------------------

func TestE2E_ListTools_ReturnsAllRegistered(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "list-tools", nil)

	for _, id := range []string{"greet", "mutate", "slow-greet"} {
		if !strings.Contains(text, id) {
			t.Errorf("expected %q in list-tools output, got: %s", id, text)
		}
	}
}

func TestE2E_ListTools_IsJSONArray(t *testing.T) {
	_, deps := buildTestServer(t, false)
	text := callTool(t, deps, "list-tools", nil)

	var arr []interface{}
	if err := json.Unmarshal([]byte(text), &arr); err != nil {
		t.Errorf("list-tools output is not a valid JSON array: %v\nbody: %s", err, text)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 tools, got %d", len(arr))
	}
}

// ---- get-tool-details ---------------------------------------------------

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
	assertCallIsError(t, deps, "get-tool-details", map[string]interface{}{"tool_id": "nonexistent"})
}

// ---- execute-tool synchronous -------------------------------------------

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

func TestE2E_ExecuteTool_Sync_WriteTool_Allowed(t *testing.T) {
	_, deps := buildTestServer(t, false) // read-only=false
	text := callTool(t, deps, "execute-tool", map[string]interface{}{
		"tool_id": "mutate",
	})
	if !strings.Contains(text, "mutation done") {
		t.Errorf("expected mutation result, got: %s", text)
	}
}

func TestE2E_ExecuteTool_Sync_WriteTool_BlockedOnReadOnlyServer(t *testing.T) {
	_, deps := buildTestServer(t, true) // read-only=true
	assertCallIsError(t, deps, "execute-tool", map[string]interface{}{"tool_id": "mutate"})
}

// ---- execute-tool asynchronous ------------------------------------------

func TestE2E_ExecuteTool_Async_FastTool(t *testing.T) {
	_, deps := buildTestServer(t, false)

	submitText := callTool(t, deps, "execute-tool", map[string]interface{}{
		"tool_id":    "greet",
		"async":      true,
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
	if jobResp["status"] != string(tools.JobStatusPending) {
		t.Errorf("expected status=pending, got %q", jobResp["status"])
	}

	// Poll until the raw result comes back.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		statusText := callTool(t, deps, "get-tool-status", map[string]interface{}{"job_id": jobID})
		if strings.Contains(statusText, "Hello, Async!") {
			return
		}
		var view map[string]interface{}
		if err := json.Unmarshal([]byte(statusText), &view); err == nil {
			switch view["status"] {
			case string(tools.JobStatusCompleted):
				return
			case string(tools.JobStatusFailed):
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

	// First poll is very likely to be pending/running because the handler sleeps 100ms.
	firstStatus := callTool(t, deps, "get-tool-status", map[string]interface{}{"job_id": jobID})
	var firstView map[string]interface{}
	json.Unmarshal([]byte(firstStatus), &firstView) //nolint:errcheck
	if firstView["status"] == string(tools.JobStatusCompleted) {
		t.Log("warning: slow-greet completed before first poll (very fast machine?)")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		text := callTool(t, deps, "get-tool-status", map[string]interface{}{"job_id": jobID})
		if strings.Contains(text, "slow hello") {
			return
		}
		var v map[string]interface{}
		if json.Unmarshal([]byte(text), &v) == nil && v["status"] == string(tools.JobStatusCompleted) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("slow-greet did not complete within 5s")
}

// ---- get-tool-status edge cases -----------------------------------------

func TestE2E_GetToolStatus_UnknownJobID(t *testing.T) {
	_, deps := buildTestServer(t, false)
	assertCallIsError(t, deps, "get-tool-status", map[string]interface{}{"job_id": "does-not-exist"})
}

// ---- concurrency --------------------------------------------------------

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

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		allDone := true
		for _, id := range jobIDs {
			view, err := deps.Jobs.Get(id)
			if err != nil || (view.Status != tools.JobStatusCompleted && view.Status != tools.JobStatusFailed) {
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

// extractText marshals result to JSON and pulls the first text content block.
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

// assertCallIsError invokes a meta-tool handler directly and asserts
// the response carries isError=true.
func assertCallIsError(t *testing.T, deps *tools.Dependencies, toolName string, args map[string]interface{}) {
	t.Helper()

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = args

	var result *mcpgo.CallToolResult
	switch toolName {
	case "list-tools":
		result, _ = tools.ListToolsHandler(deps)(context.Background(), req)
	case "get-tool-details":
		result, _ = tools.GetToolDetailsHandler(deps)(context.Background(), req)
	case "execute-tool":
		result, _ = tools.ExecuteToolHandler(deps)(context.Background(), req)
	case "get-tool-status":
		result, _ = tools.GetToolStatusHandler(deps)(context.Background(), req)
	default:
		t.Fatalf("assertCallIsError: unsupported tool %q", toolName)
	}

	data, _ := json.Marshal(result)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck

	if isErr, _ := m["isError"].(bool); !isErr {
		t.Errorf("[%s] expected isError=true, got: %s", toolName, data)
	}
}
