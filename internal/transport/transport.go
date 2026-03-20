// Package transport defines the Transport interface shared by all MCP
// transport implementations (stdio, HTTP/HTTPS, …).
package transport

import "context"

// Transport is the common interface every concrete transport must satisfy.
//
// Serve blocks until the server stops or returns an error.
// Shutdown attempts a graceful stop within the supplied context deadline.
type Transport interface {
	Serve() error
	Shutdown(ctx context.Context) error
}
