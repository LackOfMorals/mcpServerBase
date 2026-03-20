// Package tools contains the tool definition model, registry, async job
// engine, MCP meta-tool specs, and meta-tool handlers.
//
// Consuming code (the server, project-specific tools, tests) imports this
// package rather than reaching into the server package.
package tools

import (
	"context"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// ToolType categorises a registered tool operation.
type ToolType string

const (
	ToolTypeList   ToolType = "list"
	ToolTypeRead   ToolType = "read"
	ToolTypeCreate ToolType = "create"
	ToolTypeUpdate ToolType = "update"
	ToolTypeDelete ToolType = "delete"
)

// ToolHandler is the function signature every registered tool must implement.
type ToolHandler func(ctx context.Context, parameters map[string]interface{}, deps *Dependencies) (*mcp.CallToolResult, error)

// ToolDef describes a single registered tool operation.
// Named ToolDef (not Tool) to avoid collision with mcp.Tool from mcp-go.
type ToolDef struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        ToolType               `json:"type"`
	ReadOnly    bool                   `json:"readonly"`
	Parameters  []ToolParam            `json:"parameters,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Handler     ToolHandler            `json:"-"` // not serialised
}

// ToolParam describes a single parameter accepted by a ToolDef.
type ToolParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolSummary is the lightweight view of a ToolDef returned by list-tools.
type ToolSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        ToolType `json:"type"`
	ReadOnly    bool     `json:"readonly"`
}

// Dependencies bundles everything a ToolHandler (or meta-tool handler) may need.
// It is defined here so that ToolHandler, ToolRegistry, and JobRegistry can all
// reference it without creating an import cycle with the server package.
type Dependencies struct {
	Config *config.Config
	Tools  *ToolRegistry
	Jobs   *JobRegistry
	Server *mcpgoserver.MCPServer // for progress notifications
}
