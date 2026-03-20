package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// osExit can be replaced in tests.
var osExit = os.Exit

const helpText = `mcp-base — A Model Context Protocol Server

Usage:
  mcp-base [OPTIONS]

Options:
  -h, --help      Show this help message
  -v, --version   Show version information

General options:
  --read-only     Enable read-only mode: true|false  (env: READ_ONLY, default: true)
  --log-level     Log level: debug|info|notice|warning|error (env: LOG_LEVEL, default: info)
  --log-format    Log format: text|json              (env: LOG_FORMAT, default: text)

Transport:
  --transport     Transport mode: stdio|http         (env: TRANSPORT, default: stdio)

HTTP transport options (only used when --transport=http):
  --http-addr     Listen address and port            (env: HTTP_ADDR, default: 127.0.0.1:6666)
  --api-key       API key required for authenticated requests
                                                     (env: API_KEY)

TLS options (only used when --transport=http):
  --tls           Enable HTTPS: true|false           (env: TLS, default: false)
  --tls-cert      Path to TLS certificate file       (env: TLS_CERT_FILE)
  --tls-key       Path to TLS private key file       (env: TLS_KEY_FILE)

Notes:
  • CLI flags take precedence over environment variables.
  • In HTTP mode, the MCP endpoint is available at <addr>/mcp.
  • The following MCP methods do not require authentication in HTTP mode:
      initialize, notifications/initialized, ping, tools/list

Examples:
  # stdio mode (default)
  mcp-base

  # HTTP mode with API key
  mcp-base --transport=http --api-key=secret

  # HTTPS mode
  mcp-base --transport=http --tls --tls-cert=/etc/certs/server.crt --tls-key=/etc/certs/server.key --api-key=secret

  # Custom address
  mcp-base --transport=http --http-addr=0.0.0.0:8080 --api-key=secret
`

// knownValueFlags lists flags that accept a value argument.  HandleArgs skips
// these so they can be re-parsed by the standard flag package.
var knownValueFlags = []string{
	"--read-only",
	"--log-level",
	"--log-format",
	"--transport",
	"--http-addr",
	"--api-key",
	"--tls",
	"--tls-cert",
	"--tls-key",
}

// Args holds configuration values parsed from command-line flags.
type Args struct {
	ReadOnly    string
	LogLevel    string
	LogFormat   string
	Transport   string
	HTTPAddr    string
	TLSEnabled  string
	TLSCertFile string
	TLSKeyFile  string
	APIKey      string
}

// ParseConfigFlags parses CLI flags and returns configuration values.
// Must be called after HandleArgs so -h / -v are already handled.
func ParseConfigFlags() *Args {
	readOnly := flag.String("read-only", "", "Enable read-only mode: true|false (overrides READ_ONLY)")
	logLevel := flag.String("log-level", "", "Log level (overrides LOG_LEVEL)")
	logFormat := flag.String("log-format", "", "Log format (overrides LOG_FORMAT)")
	transport := flag.String("transport", "", "Transport mode: stdio|http (overrides TRANSPORT)")
	httpAddr := flag.String("http-addr", "", "HTTP listen address (overrides HTTP_ADDR)")
	tls := flag.String("tls", "", "Enable TLS: true|false (overrides TLS)")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file (overrides TLS_CERT_FILE)")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file (overrides TLS_KEY_FILE)")
	apiKey := flag.String("api-key", "", "API key for HTTP authentication (overrides API_KEY)")

	flag.Parse()

	return &Args{
		ReadOnly:    *readOnly,
		LogLevel:    *logLevel,
		LogFormat:   *logFormat,
		Transport:   *transport,
		HTTPAddr:    *httpAddr,
		TLSEnabled:  *tls,
		TLSCertFile: *tlsCert,
		TLSKeyFile:  *tlsKey,
		APIKey:      *apiKey,
	}
}

// HandleArgs processes -h/--help and -v/--version before the standard flag
// package runs.  Known configuration flags are skipped so they can be
// re-parsed later.  Unknown flags cause a usage error and exit.
func HandleArgs(version string) {
	if len(os.Args) <= 1 {
		return
	}

	flags := make(map[string]bool)
	var err error
	i := 1

	for i < len(os.Args) {
		arg := os.Args[i]
		switch arg {
		case "-h", "--help":
			flags["help"] = true
			i++
		case "-v", "--version":
			flags["version"] = true
			i++
		default:
			if arg == "--" {
				i = len(os.Args)
				continue
			}

			// Skip known value flags (and their value argument).
			if isKnownValueFlag(arg) {
				if i+1 >= len(os.Args) {
					err = fmt.Errorf("%s requires a value", arg)
					break
				}
				nextArg := os.Args[i+1]
				if strings.HasPrefix(nextArg, "-") {
					// Support --flag=value syntax: no next token needed.
					if !strings.Contains(arg, "=") {
						err = fmt.Errorf("%s requires a value (got flag %s instead)", arg, nextArg)
						break
					}
				}
				i += 2
				continue
			}

			// Support --flag=value syntax for known flags.
			if isKnownValueFlagWithEquals(arg) {
				i++
				continue
			}

			err = fmt.Errorf("unknown flag or argument: %s", arg)
			i++
		}

		if err != nil {
			break
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		osExit(1)
	}

	if flags["help"] {
		fmt.Print(helpText)
		osExit(0)
	}

	if flags["version"] {
		fmt.Printf("mcp-base version: %s\n", version)
		osExit(0)
	}
}

// isKnownValueFlag returns true if arg exactly matches a known flag name.
func isKnownValueFlag(arg string) bool {
	for _, f := range knownValueFlags {
		if arg == f {
			return true
		}
	}
	return false
}

// isKnownValueFlagWithEquals returns true if arg is a known flag in the form
// --flag=value (the standard flag package handles the parsing).
func isKnownValueFlagWithEquals(arg string) bool {
	for _, f := range knownValueFlags {
		if strings.HasPrefix(arg, f+"=") {
			return true
		}
	}
	return false
}
