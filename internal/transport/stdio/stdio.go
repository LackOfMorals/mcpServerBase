// Package stdio implements the MCP stdio transport.
// It is a thin wrapper around mcpgoserver.ServeStdio that satisfies the
// transport.Transport interface.
package stdio

import (
	"context"
	"log/slog"

	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// Transport wraps the mcp-go stdio serving function.
type Transport struct {
	mcpServer *mcpgoserver.MCPServer
}

// New creates a stdio Transport for the given MCPServer.
func New(mcpServer *mcpgoserver.MCPServer) *Transport {
	return &Transport{mcpServer: mcpServer}
}

// Serve blocks, reading JSON-RPC messages from stdin and writing responses to
// stdout. It returns when stdin is closed or an unrecoverable error occurs.
func (t *Transport) Serve() error {
	slog.Info("Transport: stdio — listening on stdin/stdout")
	return mcpgoserver.ServeStdio(t.mcpServer)
}

// Shutdown is a no-op for stdio: the process lifecycle ends naturally when
// stdin closes. Satisfies the transport.Transport interface.
func (t *Transport) Shutdown(_ context.Context) error {
	return nil
}
