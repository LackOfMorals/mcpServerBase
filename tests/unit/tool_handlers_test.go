package unit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/LackOfMorals/mcpServerBase/internal/tools"
)

// ---- list-tools handler -------------------------------------------------

func TestListToolsHandler_EmptyRegistry(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.ListToolsHandler(deps)

	result, err := handler(context.Background(), makeCallRequest(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := extractText(t, result)
	if !strings.Contains(text, "[]") {
		t.Errorf("expected empty JSON array, got: %s", text)
	}
}

func TestListToolsHandler_PopulatedRegistry(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("tool-a", true, echoHandler))
	deps.Tools.Register(sampleTool("tool-b", false, echoHandler))

	handler := tools.ListToolsHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(nil))

	text := extractText(t, result)
	if !strings.Contains(text, "tool-a") {
		t.Errorf("expected 'tool-a' in output, got: %s", text)
	}
	if !strings.Contains(text, "tool-b") {
		t.Errorf("expected 'tool-b' in output, got: %s", text)
	}
}

// ---- get-tool-details handler -------------------------------------------

func TestGetToolDetailsHandler_ValidID(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("detail-target", true, echoHandler))

	handler := tools.GetToolDetailsHandler(deps)
	result, err := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id": "detail-target",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := extractText(t, result)
	if !strings.Contains(text, "detail-target") {
		t.Errorf("expected tool ID in details output, got: %s", text)
	}
	if !strings.Contains(text, "parameters") {
		t.Errorf("expected 'parameters' field in details output, got: %s", text)
	}
}

func TestGetToolDetailsHandler_UnknownID(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.GetToolDetailsHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id": "ghost",
	}))
	assertMCPError(t, result, "unknown tool")
}

func TestGetToolDetailsHandler_MissingParam(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.GetToolDetailsHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{}))
	assertMCPError(t, result, "missing tool_id")
}

func TestGetToolDetailsHandler_BadArgFormat(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.GetToolDetailsHandler(deps)
	req := makeCallRequest(nil)
	req.Params.Arguments = "not-a-map"
	result, _ := handler(context.Background(), req)
	assertMCPError(t, result, "bad args")
}

// ---- execute-tool handler (synchronous path) ----------------------------

func TestExecuteToolHandler_Sync_Success(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("sync-tool", true, echoHandler))

	handler := tools.ExecuteToolHandler(deps)
	result, err := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id":    "sync-tool",
		"parameters": map[string]interface{}{"q": "hello"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected success result")
	}
	if !strings.Contains(extractText(t, result), "echo") {
		t.Errorf("expected echo output")
	}
}

func TestExecuteToolHandler_Sync_UnknownTool(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.ExecuteToolHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id": "not-registered",
	}))
	assertMCPError(t, result, "unknown tool")
}

func TestExecuteToolHandler_Sync_BadParameters(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("p-tool", true, echoHandler))
	handler := tools.ExecuteToolHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id":    "p-tool",
		"parameters": "not-an-object",
	}))
	assertMCPError(t, result, "bad parameters type")
}

func TestExecuteToolHandler_Sync_MissingToolID(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.ExecuteToolHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{}))
	assertMCPError(t, result, "missing tool_id")
}

// ---- execute-tool handler (asynchronous path) ---------------------------

func TestExecuteToolHandler_Async_ReturnsJobID(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("async-tool", true, echoHandler))

	handler := tools.ExecuteToolHandler(deps)
	result, err := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id": "async-tool",
		"async":   true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := extractText(t, result)
	var resp map[string]string
	if jsonErr := json.Unmarshal([]byte(text), &resp); jsonErr != nil {
		t.Fatalf("response is not valid JSON: %v — body: %s", jsonErr, text)
	}
	if resp["job_id"] == "" {
		t.Error("expected non-empty job_id in async response")
	}
	if resp["status"] != string(tools.JobStatusPending) {
		t.Errorf("expected status=%q, got %q", tools.JobStatusPending, resp["status"])
	}
}

func TestExecuteToolHandler_Async_FalseIsSync(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("sync-default", true, echoHandler))

	handler := tools.ExecuteToolHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id": "sync-default",
		"async":   false,
	}))
	text := extractText(t, result)
	if strings.Contains(text, "job_id") {
		t.Errorf("async=false should not return a job_id, got: %s", text)
	}
}

// ---- get-tool-status handler --------------------------------------------

func TestGetToolStatusHandler_UnknownJobID(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.GetToolStatusHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{
		"job_id": "no-such-job",
	}))
	assertMCPError(t, result, "unknown job")
}

func TestGetToolStatusHandler_MissingJobID(t *testing.T) {
	deps := newDeps(nil)
	handler := tools.GetToolStatusHandler(deps)
	result, _ := handler(context.Background(), makeCallRequest(map[string]interface{}{}))
	assertMCPError(t, result, "missing job_id")
}

func TestGetToolStatusHandler_EventualCompletion(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("poll-tool", true, echoHandler))

	execHandler := tools.ExecuteToolHandler(deps)
	execResult, _ := execHandler(context.Background(), makeCallRequest(map[string]interface{}{
		"tool_id": "poll-tool",
		"async":   true,
	}))
	var jobResp map[string]string
	json.Unmarshal([]byte(extractText(t, execResult)), &jobResp) //nolint:errcheck
	jobID := jobResp["job_id"]

	statusHandler := tools.GetToolStatusHandler(deps)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		statusResult, _ := statusHandler(context.Background(), makeCallRequest(map[string]interface{}{
			"job_id": jobID,
		}))
		text := extractText(t, statusResult)
		if strings.Contains(text, "echo:") {
			return
		}
		var view map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(text), &view); jsonErr == nil {
			if view["status"] == string(tools.JobStatusCompleted) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not reach completed status within timeout")
}

// ---- helpers ------------------------------------------------------------

func extractText(t *testing.T, result interface{}) string {
	t.Helper()
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
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

func assertMCPError(t *testing.T, result interface{}, label string) {
	t.Helper()
	data, _ := json.Marshal(result)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck
	isErr, _ := m["isError"].(bool)
	if !isErr {
		t.Errorf("[%s] expected isError=true, result JSON: %s", label, data)
	}
}
