package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
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
