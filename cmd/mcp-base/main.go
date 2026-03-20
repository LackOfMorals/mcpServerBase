package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/LackOfMorals/mcpServerBase/internal/cli"
	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/logger"
	"github.com/LackOfMorals/mcpServerBase/internal/server"
)

// go build -C cmd/mcp-base -o ../../bin/ -ldflags "-X 'main.Version=9999'"
var Version = "development"

func main() {
	// Handle CLI arguments (version, help, etc.)
	cli.HandleArgs(Version)

	// Parse CLI flags for configuration
	cliArgs := cli.ParseConfigFlags()

	// Load and validate configuration (env vars + CLI overrides)
	cfg, err := config.LoadConfig(&config.CLIOverrides{
		ReadOnly:  cliArgs.ReadOnly,
		LogLevel:  cliArgs.LogLevel,
		LogFormat: cliArgs.LogFormat,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load configuration: "+err.Error())
		os.Exit(1)
	}

	// Initialise global logger
	logger.Init(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg)

	// Register project-specific tools here before calling Start.
	// Example:
	//
	//   mcpServer.RegisterTool(&server.ToolDef{
	//       ID:       "my-tool",
	//       Name:     "My Tool",
	//       Type:     server.ToolTypeRead,
	//       ReadOnly: true,
	//       Handler:  myToolHandler,
	//   })
	_ = mcpServer // suppress unused warning until tools are registered

	// Gracefully handle shutdown
	defer func() {
		if err := mcpServer.Stop(); err != nil {
			slog.Error("Error stopping server", "error", err)
		}
	}()

	// Start the server (blocks until the server is stopped)
	if err := mcpServer.Start(); err != nil {
		slog.Error("Server error", "error", err)
		return // so that defer can run
	}
}
