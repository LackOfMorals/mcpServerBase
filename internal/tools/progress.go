// progress.go — helpers for sending MCP notifications/progress to the client
// during long-running tool handlers.
//
// The MCP spec allows a server to send out-of-band progress notifications
// while a tool call is still in-flight.  The client attaches a progressToken
// in _meta when it wants these; we check for it and fire if present.
//
// If the client did not supply a token (older clients, or clients that opted
// out) the helpers are no-ops — the tool still executes, just silently.

package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// ProgressSender holds the context and server reference needed to send
// notifications/progress, plus the token that ties them to the original request.
// Exported so that tool implementations in other packages can use it.
type ProgressSender struct {
	ctx    context.Context
	srv    *mcpgoserver.MCPServer
	token  mcp.ProgressToken
	active bool // false when no token was provided
}

// NewProgressSender constructs a ProgressSender from the tool call request.
// If the client did not include a progressToken the sender is inert.
func NewProgressSender(ctx context.Context, srv *mcpgoserver.MCPServer, req mcp.CallToolRequest) *ProgressSender {
	ps := &ProgressSender{ctx: ctx, srv: srv}

	if req.Params.Meta != nil && req.Params.Meta.ProgressToken != nil {
		ps.token = req.Params.Meta.ProgressToken
		ps.active = true
	}

	return ps
}

// Send fires a notifications/progress message.
// progress is 0.0–1.0 for determinate progress, or -1 for indeterminate.
// message is a human-readable status line for the client/LLM to display.
func (ps *ProgressSender) Send(progress float64, message string) {
	if !ps.active || ps.srv == nil {
		return
	}

	// SendNotificationToClient is best-effort; never fail the tool over it.
	_ = ps.srv.SendNotificationToClient(ps.ctx, "notifications/progress", map[string]any{
		"progressToken": ps.token,
		"progress":      progress,
		"message":       message,
	})
}

// Statusf formats a message and sends with an indeterminate progress value (-1).
func (ps *ProgressSender) Statusf(format string, args ...any) {
	ps.Send(-1, fmt.Sprintf(format, args...))
}

// Step sends a discrete step out of a known total (e.g. step 2 of 4).
func (ps *ProgressSender) Step(current, total int, message string) {
	var pct float64
	if total > 0 {
		pct = float64(current) / float64(total)
	}
	ps.Send(pct, fmt.Sprintf("[%d/%d] %s", current, total, message))
}
