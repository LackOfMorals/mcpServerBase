// Package unit contains unit tests for the internal/server package.
package unit

import (
	"context"
	"fmt"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/server"
	"github.com/mark3labs/mcp-go/mcp"
)

// ---- helpers shared across the unit tests --------------------------------

// newDeps builds a minimal *server.Dependencies wired to a fresh registry
// and job store but with a nil MCPServer (fine for unit tests).
func newDeps(cfg *config.Config) *server.Dependencies {
	if cfg == nil {
		cfg = &config.Config{ReadOnly: false}
	}
	return &server.Dependencies{
		Config: cfg,
		Tools:  server.NewToolRegistry(),
		Jobs:   server.NewJobRegistry(),
		Server: nil, // not needed for handler/registry unit tests
	}
}

// echoHandler is a simple ToolHandler that echoes its parameters as JSON text.
func echoHandler(_ context.Context, params map[string]interface{}, _ *server.Dependencies) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf("echo: %v", params)), nil
}

// failHandler always returns a Go-level error (not an MCP error result).
func failHandler(_ context.Context, _ map[string]interface{}, _ *server.Dependencies) (*mcp.CallToolResult, error) {
	return nil, fmt.Errorf("handler exploded")
}

// slowHandler blocks until its context is cancelled, simulating a long-running tool.
func slowHandler(ctx context.Context, _ map[string]interface{}, _ *server.Dependencies) (*mcp.CallToolResult, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// sampleTool returns a fully-populated *server.ToolDef for use in multiple tests.
func sampleTool(id string, readOnly bool, h server.ToolHandler) *server.ToolDef {
	return &server.ToolDef{
		ID:       id,
		Name:     "Sample " + id,
		Type:     server.ToolTypeRead,
		ReadOnly: readOnly,
		Parameters: []server.ToolParam{
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
