// handlers.go — MCP tool handlers for the four meta-tools and the
// GetAllMetaTools helper used by the server to register them.

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// GetAllMetaTools returns the four meta-tool ServerTool entries wired to the
// supplied Dependencies.  Call this once during server startup and pass the
// result to MCPServer.AddTools.
func GetAllMetaTools(deps *Dependencies) []mcpgoserver.ServerTool {
	return []mcpgoserver.ServerTool{
		{Tool: ListToolsSpec(), Handler: ListToolsHandler(deps)},
		{Tool: GetToolDetailsSpec(), Handler: GetToolDetailsHandler(deps)},
		{Tool: ExecuteToolSpec(), Handler: ExecuteToolHandler(deps)},
		{Tool: GetToolStatusSpec(), Handler: GetToolStatusHandler(deps)},
	}
}

// ListToolsHandler returns all registered tool summaries as JSON.
func ListToolsHandler(deps *Dependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		summaries := deps.Tools.GetAllSummaries()
		data, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialise tool list: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetToolDetailsHandler returns the full ToolDef (including parameters) for a
// given tool_id.
func GetToolDetailsHandler(deps *Dependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}

		toolID, ok := args["tool_id"].(string)
		if !ok || toolID == "" {
			return mcp.NewToolResultError("tool_id parameter is required and must be a string"), nil
		}

		tool, err := deps.Tools.GetTool(toolID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(tool, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialise tool details: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// ExecuteToolHandler runs a registered tool, either synchronously (default) or
// asynchronously when async=true is present in the arguments.
//
// Synchronous:  blocks and returns the tool result directly.
// Asynchronous: submits the job, returns {"job_id":"…","status":"pending"}.
func ExecuteToolHandler(deps *Dependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}

		toolID, ok := args["tool_id"].(string)
		if !ok || toolID == "" {
			return mcp.NewToolResultError("tool_id parameter is required and must be a string"), nil
		}

		// Extract optional parameters map.
		parameters := make(map[string]interface{})
		if paramsVal, exists := args["parameters"]; exists {
			if params, ok := paramsVal.(map[string]interface{}); ok {
				parameters = params
			} else {
				return mcp.NewToolResultError("parameters must be an object"), nil
			}
		}

		// Determine execution mode.
		asyncMode := false
		if asyncVal, exists := args["async"]; exists {
			if b, ok := asyncVal.(bool); ok {
				asyncMode = b
			}
		}

		if asyncMode {
			jobID, err := deps.Tools.AsyncExecuteTool(ctx, toolID, parameters, deps)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]string{
				"job_id": jobID,
				"status": string(JobStatusPending),
			})
			return mcp.NewToolResultText(string(data)), nil
		}

		// Synchronous path.
		return deps.Tools.ExecuteTool(ctx, toolID, parameters, deps)
	}
}

// GetToolStatusHandler polls the JobRegistry for the current state of an async
// job. When the job has completed it returns the raw tool result inline so the
// caller receives identical content to a synchronous execution.
func GetToolStatusHandler(deps *Dependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}

		jobID, ok := args["job_id"].(string)
		if !ok || jobID == "" {
			return mcp.NewToolResultError("job_id parameter is required and must be a string"), nil
		}

		view, err := deps.Jobs.Get(jobID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// When the job has completed, return the raw *mcp.CallToolResult so the
		// LLM receives the same payload it would get from a synchronous call.
		if view.Status == JobStatusCompleted {
			result, err := deps.Jobs.GetResult(jobID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if result != nil {
				return result, nil
			}
		}

		// Job is still pending/running (or completed with no result) — return the status view.
		data, err := json.MarshalIndent(view, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialise job status: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
