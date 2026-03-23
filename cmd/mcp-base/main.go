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
	cli.HandleArgs(Version)

	cliArgs := cli.ParseConfigFlags()

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
		fmt.Fprintln(os.Stderr, "configuration error: "+err.Error())
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	mcpServer := server.New(Version, cfg)
	// Register all of the tools that are described in /internnal/tools/catalog.go.
	// catalog.go has instructions on how to do that

	tools.RegisterAll(mcpServer)

	defer func() {
		if err := mcpServer.Stop(); err != nil {
			slog.Error("error stopping server", "error", err)
		}
	}()

	if err := mcpServer.Start(); err != nil {
		slog.Error("server error", "error", err)
	}
}
