// tool_registry.go
//
// ToolRegistry manages all ToolDef registrations and dispatches execution.
// It replaces the previous OutcomeRegistry.

package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolRegistry manages all available ToolDefs.
type ToolRegistry struct {
	tools map[string]*ToolDef
}

// NewToolRegistry creates an empty ToolRegistry.
// Add tools with Register before starting the server.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDef),
	}
}

// Register adds a ToolDef to the registry.
// Panics if the ID has already been registered to catch mis-configuration early.
func (r *ToolRegistry) Register(t *ToolDef) {
	if _, exists := r.tools[t.ID]; exists {
		panic(fmt.Sprintf("tool ID already registered: %s", t.ID))
	}
	r.tools[t.ID] = t
}

// GetAllSummaries returns a ToolSummary for every registered tool.
func (r *ToolRegistry) GetAllSummaries() []ToolSummary {
	summaries := make([]ToolSummary, 0, len(r.tools))
	for _, t := range r.tools {
		summaries = append(summaries, ToolSummary{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Type:        t.Type,
			ReadOnly:    t.ReadOnly,
		})
	}
	return summaries
}

// GetTool returns the full ToolDef for the given ID.
func (r *ToolRegistry) GetTool(id string) (*ToolDef, error) {
	t, exists := r.tools[id]
	if !exists {
		return nil, fmt.Errorf("tool with ID %q not found", id)
	}
	return t, nil
}

// ExecuteTool runs the handler for the given tool ID synchronously.
// Returns an MCP error result (not a Go error) for user-facing problems such as
// unknown tool, read-only violation, or missing handler.
func (r *ToolRegistry) ExecuteTool(ctx context.Context, id string, parameters map[string]interface{}, deps *Dependencies) (*mcp.CallToolResult, error) {
	t, err := r.GetTool(id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !t.ReadOnly && deps.Config != nil && deps.Config.ReadOnly {
		return mcp.NewToolResultError(fmt.Sprintf(
			"cannot execute %q: server is in read-only mode. Set READ_ONLY=false to enable write operations.", id,
		)), nil
	}

	if t.Handler == nil {
		return mcp.NewToolResultError(fmt.Sprintf("no handler registered for tool: %s", id)), nil
	}

	return t.Handler(ctx, parameters, deps)
}

// AsyncExecuteTool validates the tool then submits it to the JobRegistry for
// background execution. Returns the job ID or an error if the tool cannot run.
func (r *ToolRegistry) AsyncExecuteTool(ctx context.Context, id string, parameters map[string]interface{}, deps *Dependencies) (string, error) {
	t, err := r.GetTool(id)
	if err != nil {
		return "", err
	}

	if !t.ReadOnly && deps.Config != nil && deps.Config.ReadOnly {
		return "", fmt.Errorf(
			"cannot execute %q: server is in read-only mode. Set READ_ONLY=false to enable write operations.", id,
		)
	}

	if t.Handler == nil {
		return "", fmt.Errorf("no handler registered for tool: %s", id)
	}

	jobID := deps.Jobs.Submit(ctx, id, t.Handler, parameters, deps)
	return jobID, nil
}
