package unit

import (
	"context"
	"testing"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/tools"
)

// ---- Register -----------------------------------------------------------

func TestToolRegistry_Register_Success(t *testing.T) {
	r := tools.NewToolRegistry()
	r.Register(sampleTool("t1", true, echoHandler))

	summaries := r.GetAllSummaries()
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].ID != "t1" {
		t.Errorf("expected ID 't1', got %q", summaries[0].ID)
	}
}

func TestToolRegistry_Register_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration, got none")
		}
	}()
	r := tools.NewToolRegistry()
	r.Register(sampleTool("dup", true, echoHandler))
	r.Register(sampleTool("dup", true, echoHandler)) // must panic
}

// ---- GetAllSummaries ----------------------------------------------------

func TestToolRegistry_GetAllSummaries_Empty(t *testing.T) {
	r := tools.NewToolRegistry()
	if s := r.GetAllSummaries(); len(s) != 0 {
		t.Errorf("expected empty slice, got %d items", len(s))
	}
}

func TestToolRegistry_GetAllSummaries_MultipleTools(t *testing.T) {
	r := tools.NewToolRegistry()
	r.Register(sampleTool("a", true, echoHandler))
	r.Register(sampleTool("b", false, echoHandler))
	r.Register(sampleTool("c", true, echoHandler))

	summaries := r.GetAllSummaries()
	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}
}

func TestToolRegistry_GetAllSummaries_ReadOnlyField(t *testing.T) {
	r := tools.NewToolRegistry()
	r.Register(sampleTool("rw", false, echoHandler))

	s := r.GetAllSummaries()[0]
	if s.ReadOnly {
		t.Error("expected ReadOnly=false")
	}
}

// ---- GetTool ------------------------------------------------------------

func TestToolRegistry_GetTool_Found(t *testing.T) {
	r := tools.NewToolRegistry()
	r.Register(sampleTool("find-me", true, echoHandler))

	tool, err := r.GetTool("find-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.ID != "find-me" {
		t.Errorf("expected ID 'find-me', got %q", tool.ID)
	}
}

func TestToolRegistry_GetTool_NotFound(t *testing.T) {
	r := tools.NewToolRegistry()
	_, err := r.GetTool("ghost")
	if err == nil {
		t.Fatal("expected an error for unknown tool, got nil")
	}
}

// ---- ExecuteTool --------------------------------------------------------

func TestToolRegistry_ExecuteTool_Sync_Success(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("echo", true, echoHandler))

	result, err := deps.Tools.ExecuteTool(context.Background(), "echo", map[string]interface{}{"q": "hello"}, deps)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Errorf("expected success result, got error result")
	}
}

func TestToolRegistry_ExecuteTool_UnknownTool(t *testing.T) {
	deps := newDeps(nil)
	result, err := deps.Tools.ExecuteTool(context.Background(), "no-such-tool", nil, deps)
	if err != nil {
		t.Fatalf("unexpected Go error (should be MCP error result): %v", err)
	}
	if !result.IsError {
		t.Error("expected MCP error result for unknown tool")
	}
}

func TestToolRegistry_ExecuteTool_ReadOnlyBlock(t *testing.T) {
	cfg := &config.Config{ReadOnly: true}
	deps := newDeps(cfg)
	deps.Tools.Register(sampleTool("write-op", false, echoHandler))

	result, err := deps.Tools.ExecuteTool(context.Background(), "write-op", nil, deps)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected MCP error result when server is read-only")
	}
}

func TestToolRegistry_ExecuteTool_ReadOnlyToolAllowed(t *testing.T) {
	cfg := &config.Config{ReadOnly: true}
	deps := newDeps(cfg)
	deps.Tools.Register(sampleTool("read-op", true, echoHandler))

	result, err := deps.Tools.ExecuteTool(context.Background(), "read-op", nil, deps)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Error("read-only tool should succeed even when server is in read-only mode")
	}
}

func TestToolRegistry_ExecuteTool_NilHandler(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(&tools.ToolDef{ID: "null-handler", ReadOnly: true, Handler: nil})

	result, err := deps.Tools.ExecuteTool(context.Background(), "null-handler", nil, deps)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected MCP error result for nil handler")
	}
}

// ---- AsyncExecuteTool ---------------------------------------------------

func TestToolRegistry_AsyncExecuteTool_ReturnsJobID(t *testing.T) {
	deps := newDeps(nil)
	deps.Tools.Register(sampleTool("async-echo", true, echoHandler))

	jobID, err := deps.Tools.AsyncExecuteTool(context.Background(), "async-echo", map[string]interface{}{}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobID == "" {
		t.Error("expected non-empty job ID")
	}
}

func TestToolRegistry_AsyncExecuteTool_UnknownTool(t *testing.T) {
	deps := newDeps(nil)
	_, err := deps.Tools.AsyncExecuteTool(context.Background(), "nope", nil, deps)
	if err == nil {
		t.Error("expected Go error for unknown tool, got nil")
	}
}

func TestToolRegistry_AsyncExecuteTool_ReadOnlyBlock(t *testing.T) {
	cfg := &config.Config{ReadOnly: true}
	deps := newDeps(cfg)
	deps.Tools.Register(sampleTool("write-async", false, echoHandler))

	_, err := deps.Tools.AsyncExecuteTool(context.Background(), "write-async", nil, deps)
	if err == nil {
		t.Error("expected read-only error, got nil")
	}
}
