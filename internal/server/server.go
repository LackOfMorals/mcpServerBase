package server

import (
	"log/slog"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/mark3labs/mcp-go/server"
)

// Neo4jMCPServer wraps the mcp-go server with our tool registry and job store.
type Neo4jMCPServer struct {
	MCPServer *server.MCPServer
	config    *config.Config
	tools     *ToolRegistry
	jobs      *JobRegistry
	version   string
}

// Dependencies bundles everything a tool handler might need.
type Dependencies struct {
	Config *config.Config
	Tools  *ToolRegistry
	Jobs   *JobRegistry
	Server *server.MCPServer // for progress notifications
}

// NewNeo4jMCPServer creates and configures a new MCP server instance.
// cfg must already be validated.
func NewNeo4jMCPServer(version string, cfg *config.Config) *Neo4jMCPServer {
	mcpServer := server.NewMCPServer(
		"mcp-server-name",
		version,
		server.WithToolCapabilities(true),
		server.WithInstructions("Make sure these instructions provide a good summary of what the MCP server does"),
	)

	return &Neo4jMCPServer{
		MCPServer: mcpServer,
		config:    cfg,
		version:   version,
		tools:     NewToolRegistry(),
		jobs:      NewJobRegistry(),
	}
}

// RegisterTool adds a ToolDef to the server's registry.
// Must be called before Start so the tool is available when meta-tools are wired up.
func (s *Neo4jMCPServer) RegisterTool(t *ToolDef) {
	s.tools.Register(t)
}

// Start initialises and begins serving over stdio. Blocks until the server stops.
func (s *Neo4jMCPServer) Start() error {
	slog.Info("Starting MCP Server...")

	if err := s.verifyRequirements(); err != nil {
		return err
	}

	deps := &Dependencies{
		Tools:  s.tools,
		Jobs:   s.jobs,
		Config: s.config,
		Server: s.MCPServer,
	}

	s.registerMetaTools(deps)

	slog.Info("MCP Server ready. Listening on stdio...")
	return server.ServeStdio(s.MCPServer)
}

// verifyRequirements performs any pre-flight checks needed before serving.
func (s *Neo4jMCPServer) verifyRequirements() error {
	return nil
}

// registerMetaTools adds the four fixed meta-tools to the MCP server.
func (s *Neo4jMCPServer) registerMetaTools(deps *Dependencies) {
	s.MCPServer.AddTools(GetAllMetaTools(deps)...)
}

// GetAllMetaTools returns the four meta-tool ServerTool entries.
// Exposed so tests can inspect the set without starting the server.
func GetAllMetaTools(deps *Dependencies) []server.ServerTool {
	return []server.ServerTool{
		{Tool: ListToolsSpec(), Handler: ListToolsHandler(deps)},
		{Tool: GetToolDetailsSpec(), Handler: GetToolDetailsHandler(deps)},
		{Tool: ExecuteToolSpec(), Handler: ExecuteToolHandler(deps)},
		{Tool: GetToolStatusSpec(), Handler: GetToolStatusHandler(deps)},
	}
}

// Stop performs a graceful shutdown.
func (s *Neo4jMCPServer) Stop() error {
	slog.Info("Stopping MCP Server...")
	return nil
}
