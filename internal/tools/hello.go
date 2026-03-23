// hello.go — "hello world" tool.
//
// A minimal example that demonstrates the ToolHandler signature and shows
// how to add a new tool to the server.  It takes no parameters and always
// returns the string "Hello, World!".

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// helloWorldHandler is the handler for the "hello-world" tool.
func helloWorldHandler(_ context.Context, _ map[string]interface{}, _ *Dependencies) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("Hello, World!"), nil
}
