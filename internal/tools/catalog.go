// catalog.go
//
// RegisterAll is the single place where every project-specific MCP tool is
// defined and registered with the server.
//
// To add a new tool:
//   1. Write a ToolHandler function (or add it in its own file under tools/).
//   2. Append a *ToolDef entry to the slice returned by catalog().
//
// main.go stays minimal — it only needs to call tools.RegisterAll(mcpServer).

package tools

// Registrar is the narrow interface that RegisterAll requires.
// *server.MCPServer satisfies it without tools needing to import server.
type Registrar interface {
	RegisterTool(*ToolDef)
}

// RegisterAll registers every tool in the catalog with r.
// Call this from main.go before starting the server.
func RegisterAll(r Registrar) {
	for _, def := range catalog() {
		r.RegisterTool(def)
	}
}

// catalog returns all project-specific ToolDefs.
// Add new tools here.
func catalog() []*ToolDef {
	return []*ToolDef{
		{
			ID:          "hello-world",
			Name:        "Hello World",
			Description: "Returns the string 'Hello, World!' — a minimal example tool.",
			Type:        ToolTypeRead,
			ReadOnly:    true,
			Handler:     helloWorldHandler,
		},
	}
}

// ---- tool handler implementations ---------------------------------------
//
// Each handler follows the ToolHandler signature:
//
//   func(ctx context.Context, parameters map[string]interface{}, deps *Dependencies) (*mcp.CallToolResult, error)
//
// Add handler functions below as the catalog grows, or keep them in separate
// files under this package (e.g. tools/echo.go) and reference them here.
