package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/LackOfMorals/mcpServerBase/internal/cli"
	"github.com/LackOfMorals/mcpServerBase/internal/config"
	"github.com/LackOfMorals/mcpServerBase/internal/logger"
	"github.com/LackOfMorals/mcpServerBase/internal/server"
	"github.com/LackOfMorals/mcpServerBase/internal/tools"
)

// go build -C cmd/mcp-base -o ../../bin/ -ldflags "-X 'main.Version=9999'"
var Version = "development"

func main() {
	// Handle -h/--help and -v/--version before anything else.
	cli.HandleArgs(Version)

	// Parse all configuration flags.
	cliArgs := cli.ParseConfigFlags()

	// Load and validate configuration (env vars + CLI overrides).
	cfg, err := config.LoadConfig(&config.CLIOverrides{
		ReadOnly:    cliArgs.ReadOnly,
		LogLevel:    cliArgs.LogLevel,
		LogFormat:   cliArgs.LogFormat,
		Transport:   cliArgs.Transport,
		HTTPAddr:    cliArgs.HTTPAddr,
		TLSEnabled:  cliArgs.TLSEnabled,
		TLSCertFile: cliArgs.TLSCertFile,
		TLSKeyFile:  cliArgs.TLSKeyFile,
		APIKey:      cliArgs.APIKey,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load configuration: "+err.Error())
		os.Exit(1)
	}

	// Initialise global logger.
	logger.Init(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	// Create the MCP server.
	mcpServer := server.New(Version, cfg)

	// Register project-specific tools here before calling Start.
	// Example:
	//
	//   mcpServer.RegisterTool(&tools.ToolDef{
	//       ID:       "my-tool",
	//       Name:     "My Tool",
	//       Type:     tools.ToolTypeRead,
	//       ReadOnly: true,
	//       Handler:  myToolHandler,
	//   })
	_ = tools.ToolTypeRead // satisfies the import until real tools are registered

	// Graceful shutdown on Stop.
	defer func() {
		if err := mcpServer.Stop(); err != nil {
			slog.Error("Error stopping server", "error", err)
		}
	}()

	// Start blocks until the transport exits.
	if err := mcpServer.Start(); err != nil {
		slog.Error("Server error", "error", err)
	}
}
