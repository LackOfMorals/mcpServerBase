// tool_specs.go
//
// MCP tool specifications for the four meta-tools:
//
//   list-tools          – enumerate registered tools
//   get-tool-details    – inspect a single tool's parameters
//   execute-tool        – run a tool (sync or async)
//   get-tool-status     – poll an async job

package server

import "github.com/mark3labs/mcp-go/mcp"

// ListToolsSpec returns the MCP tool spec for listing all registered tools.
func ListToolsSpec() mcp.Tool {
	return mcp.NewTool("list-tools",
		mcp.WithDescription(`List all tools registered with this MCP server.
Returns a summary of each tool including its ID, name, description, type, and whether it is read-only.
Use this to discover what operations are available before calling get-tool-details or execute-tool.`),
		mcp.WithTitleAnnotation("List Available Tools"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetToolDetailsSpec returns the MCP tool spec for fetching full tool metadata.
func GetToolDetailsSpec() mcp.Tool {
	return mcp.NewTool("get-tool-details",
		mcp.WithDescription(`Get detailed information about a specific tool, including its full parameter list and requirements.
Call list-tools first to obtain valid tool_id values.`),
		mcp.WithTitleAnnotation("Get Tool Details"),
		mcp.WithString("tool_id",
			mcp.Required(),
			mcp.Description("The ID of the tool to inspect (from list-tools)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// ExecuteToolSpec returns the MCP tool spec for executing a registered tool.
// Set async=true to return a job_id immediately and poll with get-tool-status.
func ExecuteToolSpec() mcp.Tool {
	return mcp.NewTool("execute-tool",
		mcp.WithDescription(`Execute a registered tool with the supplied parameters.

Synchronous mode (default, async=false):
  Blocks until the tool finishes and returns the result directly.

Asynchronous mode (async=true):
  Returns {"job_id": "...", "status": "pending"} immediately.
  Poll get-tool-status with the job_id until status is "completed" or "failed".

Use list-tools to discover tools and get-tool-details to learn their parameters.`),
		mcp.WithTitleAnnotation("Execute Tool"),
		mcp.WithString("tool_id",
			mcp.Required(),
			mcp.Description("The ID of the tool to execute (from list-tools)")),
		mcp.WithObject("parameters",
			mcp.Description("Parameters required by the tool (see get-tool-details for the schema)"),
			mcp.AdditionalProperties(true)),
		mcp.WithBoolean("async",
			mcp.Description("When true, execute asynchronously and return a job_id for polling. Default: false.")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

// GetToolStatusSpec returns the MCP tool spec for polling an async job.
func GetToolStatusSpec() mcp.Tool {
	return mcp.NewTool("get-tool-status",
		mcp.WithDescription(`Poll the status of an async tool execution started with execute-tool (async=true).

Possible status values:
  pending   – job queued, not yet started
  running   – handler is executing
  completed – finished successfully; result_content holds the tool output
  failed    – handler returned an error; error field describes the failure`),
		mcp.WithTitleAnnotation("Get Tool Execution Status"),
		mcp.WithString("job_id",
			mcp.Required(),
			mcp.Description("The job_id returned by execute-tool when async=true")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}
