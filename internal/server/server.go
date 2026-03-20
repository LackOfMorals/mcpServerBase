// Package server wires the MCP server lifecycle together.
// All tool types, registry logic, and handler implementations live in the
// sibling tools package; this package is responsible only for startup,
// shutdown, and exposing a RegisterTool entry-point to callers (e.g. main).
package server

import (
	"log/slog"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/tools"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// Neo4jMCPServer wraps the mcp-go server with our tool registry and job store.
type Neo4jMCPServer struct {
	MCPServer *mcpgoserver.MCPServer
	config    *config.Config
	tools     *tools.ToolRegistry
	jobs      *tools.JobRegistry
	version   string
}

// NewNeo4jMCPServer creates and configures a new MCP server instance.
// cfg must already be validated.
func NewNeo4jMCPServer(version string, cfg *config.Config) *Neo4jMCPServer {
	mcpServer := mcpgoserver.NewMCPServer(
		"mcp-server-name",
		version,
		mcpgoserver.WithToolCapabilities(true),
		mcpgoserver.WithInstructions("Make sure these instructions provide a good summary of what the MCP server does"),
	)

	return &Neo4jMCPServer{
		MCPServer: mcpServer,
		config:    cfg,
		version:   version,
		tools:     tools.NewToolRegistry(),
		jobs:      tools.NewJobRegistry(),
	}
}

// RegisterTool adds a ToolDef to the server's registry.
// Must be called before Start so the tool is available when meta-tools are wired up.
func (s *Neo4jMCPServer) RegisterTool(t *tools.ToolDef) {
	s.tools.Register(t)
}

// Start initialises and begins serving over stdio. Blocks until the server stops.
func (s *Neo4jMCPServer) Start() error {
	slog.Info("Starting MCP Server...")

	if err := s.verifyRequirements(); err != nil {
		return err
	}

	deps := &tools.Dependencies{
		Tools:  s.tools,
		Jobs:   s.jobs,
		Config: s.config,
		Server: s.MCPServer,
	}

	s.MCPServer.AddTools(tools.GetAllMetaTools(deps)...)

	slog.Info("MCP Server ready. Listening on stdio...")
	return mcpgoserver.ServeStdio(s.MCPServer)
}

// verifyRequirements performs any pre-flight checks needed before serving.
func (s *Neo4jMCPServer) verifyRequirements() error {
	return nil
}

// Stop performs a graceful shutdown.
func (s *Neo4jMCPServer) Stop() error {
	slog.Info("Stopping MCP Server...")
	return nil
}
