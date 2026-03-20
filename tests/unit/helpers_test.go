// Package unit contains unit tests for the internal/tools package.
package unit

import (
	"context"
	"fmt"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// ---- helpers shared across the unit tests --------------------------------

// newDeps builds a minimal *tools.Dependencies wired to a fresh registry
// and job store but with a nil MCPServer (fine for unit tests).
func newDeps(cfg *config.Config) *tools.Dependencies {
	if cfg == nil {
		cfg = &config.Config{ReadOnly: false}
	}
	return &tools.Dependencies{
		Config: cfg,
		Tools:  tools.NewToolRegistry(),
		Jobs:   tools.NewJobRegistry(),
		Server: nil, // not needed for handler/registry unit tests
	}
}

// echoHandler is a simple ToolHandler that echoes its parameters as text.
func echoHandler(_ context.Context, params map[string]interface{}, _ *tools.Dependencies) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf("echo: %v", params)), nil
}

// failHandler always returns a Go-level error (not an MCP error result).
func failHandler(_ context.Context, _ map[string]interface{}, _ *tools.Dependencies) (*mcp.CallToolResult, error) {
	return nil, fmt.Errorf("handler exploded")
}

// slowHandler blocks until its context is cancelled, simulating a long-running tool.
func slowHandler(ctx context.Context, _ map[string]interface{}, _ *tools.Dependencies) (*mcp.CallToolResult, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// sampleTool returns a fully-populated *tools.ToolDef for use in multiple tests.
func sampleTool(id string, readOnly bool, h tools.ToolHandler) *tools.ToolDef {
	return &tools.ToolDef{
		ID:       id,
		Name:     "Sample " + id,
		Type:     tools.ToolTypeRead,
		ReadOnly: readOnly,
		Parameters: []tools.ToolParam{
			{Name: "q", Type: "string", Description: "query", Required: true},
		},
		Handler: h,
	}
}

// makeCallRequest builds a minimal mcp.CallToolRequest whose Arguments field
// matches what the handlers expect (map[string]interface{}).
func makeCallRequest(args map[string]interface{}) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}
