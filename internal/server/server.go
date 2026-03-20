// Package server wires the MCP server lifecycle together.
//
// It creates the mcp-go MCPServer, registers all tool meta-tools, selects
// the appropriate transport (stdio or HTTP/HTTPS) based on config, and
// delegates Serve/Shutdown to that transport.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/tools"
	"github.com/LackOfMorals/mcpServerBase/internal/transport"
	httpsvr "github.com/LackOfMorals/mcpServerBase/internal/transport/http"
	"github.com/LackOfMorals/mcpServerBase/internal/transport/stdio"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// shutdownTimeout is the maximum time given to a graceful shutdown.
const shutdownTimeout = 10 * time.Second

// MCPServer wraps the mcp-go server with our tool registry, job store, and
// the active transport.
type MCPServer struct {
	MCPServer *mcpgoserver.MCPServer
	cfg       *config.Config
	tools     *tools.ToolRegistry
	jobs      *tools.JobRegistry
	version   string
	transport transport.Transport
}

// New creates and configures an MCPServer.  cfg must already be validated.
func New(version string, cfg *config.Config) *MCPServer {
	mcpSrv := mcpgoserver.NewMCPServer(
		"mcp-server-name",
		version,
		mcpgoserver.WithToolCapabilities(true),
		mcpgoserver.WithInstructions("Make sure these instructions provide a good summary of what the MCP server does"),
	)

	return &MCPServer{
		MCPServer: mcpSrv,
		cfg:       cfg,
		version:   version,
		tools:     tools.NewToolRegistry(),
		jobs:      tools.NewJobRegistry(),
	}
}

// RegisterTool adds a ToolDef to the server's registry.
// Must be called before Start.
func (s *MCPServer) RegisterTool(t *tools.ToolDef) {
	s.tools.Register(t)
}

// Start wires the tool dependencies, registers meta-tools, picks a transport,
// and begins serving. It blocks until the transport stops.
func (s *MCPServer) Start() error {
	slog.Info("Starting MCP Server...", "transport", s.cfg.Transport, "version", s.version)

	deps := &tools.Dependencies{
		Tools:  s.tools,
		Jobs:   s.jobs,
		Config: s.cfg,
		Server: s.MCPServer,
	}
	s.MCPServer.AddTools(tools.GetAllMetaTools(deps)...)

	t, err := newTransport(s.cfg, s.MCPServer)
	if err != nil {
		return fmt.Errorf("creating transport: %w", err)
	}
	s.transport = t

	return t.Serve()
}

// Stop initiates a graceful shutdown of the active transport.
func (s *MCPServer) Stop() error {
	slog.Info("Stopping MCP Server...")
	if s.transport == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.transport.Shutdown(ctx)
}

// newTransport selects and constructs the correct transport based on cfg.
func newTransport(cfg *config.Config, mcpSrv *mcpgoserver.MCPServer) (transport.Transport, error) {
	switch cfg.Transport {
	case config.TransportHTTP:
		return httpsvr.New(cfg, mcpSrv), nil
	case config.TransportStdio, "":
		return stdio.New(mcpSrv), nil
	default:
		return nil, fmt.Errorf("unknown transport %q (valid: stdio, http)", cfg.Transport)
	}
}
