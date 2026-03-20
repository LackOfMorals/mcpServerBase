// Package httpsvr implements the MCP HTTP/HTTPS transport.
//
// It wraps mcp-go's StreamableHTTPServer as an http.Handler and hosts it
// inside a standard net/http.Server, enabling:
//
//   - Plain HTTP or HTTPS (TLS) via configuration.
//   - Per-request access logging.
//   - API-key authentication with a configurable set of public MCP methods
//     that bypass the key check (initialize, notifications/initialized, ping,
//     tools/list by default).
//
// The MCP endpoint is mounted at /mcp (the StreamableHTTPServer default).
package httpsvr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// Transport is the HTTP/HTTPS MCP transport.
type Transport struct {
	cfg        *config.Config
	mcpServer  *mcpgoserver.MCPServer
	httpServer *http.Server
	streamable *mcpgoserver.StreamableHTTPServer
	mu         sync.Mutex
}

// New creates an HTTP Transport.  Call Serve() to start listening.
func New(cfg *config.Config, mcpServer *mcpgoserver.MCPServer) *Transport {
	return &Transport{cfg: cfg, mcpServer: mcpServer}
}

// Serve builds the middleware chain, starts the HTTP(S) server, and blocks
// until the server is stopped.  It satisfies the transport.Transport interface.
func (t *Transport) Serve() error {
	t.mu.Lock()

	// Create the StreamableHTTP MCP handler.
	t.streamable = mcpgoserver.NewStreamableHTTPServer(t.mcpServer)

	// Build middleware chain: logging → auth → MCP handler.
	chain := NewHandler(t.cfg.APIKey, t.cfg.HTTPPublicMethods, t.streamable)

	t.httpServer = &http.Server{
		Addr:    t.cfg.HTTPAddr,
		Handler: chain,
	}

	srv := t.httpServer
	useTLS := t.cfg.TLSEnabled
	certFile := t.cfg.TLSCertFile
	keyFile := t.cfg.TLSKeyFile
	t.mu.Unlock()

	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	slog.Info("Transport: HTTP — listening",
		"addr", t.cfg.HTTPAddr,
		"tls", useTLS,
		"url", fmt.Sprintf("%s://%s/mcp", scheme, t.cfg.HTTPAddr),
	)

	var err error
	if useTLS {
		err = srv.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = srv.ListenAndServe()
	}

	// ListenAndServe/ListenAndServeTLS return http.ErrServerClosed on graceful
	// shutdown — that is not an error from the caller's perspective.
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops the HTTP server and closes all active MCP
// sessions within the supplied context deadline.
func (t *Transport) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	httpSrv := t.httpServer
	streamable := t.streamable
	t.mu.Unlock()

	var firstErr error

	if streamable != nil {
		if err := streamable.Shutdown(ctx); err != nil {
			firstErr = fmt.Errorf("streamable shutdown: %w", err)
		}
	}
	if httpSrv != nil {
		if err := httpSrv.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("http server shutdown: %w", err)
		}
	}
	return firstErr
}
