package config

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/LackOfMorals/mcpServerBase/internal/logger"
)

// TransportStdio and TransportHTTP are the supported transport mode values.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// DefaultHTTPAddr is the listen address used when none is configured.
const DefaultHTTPAddr = "127.0.0.1:6666"

// DefaultPublicMethods is the set of MCP JSON-RPC methods that are
// accessible without API-key authentication in HTTP mode.
var DefaultPublicMethods = []string{
	"initialize",
	"notifications/initialized",
	"ping",
	"tools/list",
}

// Config holds the application configuration.
type Config struct {
	// General
	ReadOnly  bool   // Disables tools that would make changes. True by default.
	LogLevel  string // Logging level. Default: info.
	LogFormat string // Log format. Default: text.

	// Transport selection
	Transport string // "stdio" (default) or "http"

	// HTTP transport
	HTTPAddr          string   // Listen address. Default: 127.0.0.1:6666.
	TLSEnabled        bool     // Serve HTTPS when true.
	TLSCertFile       string   // Path to TLS certificate file.
	TLSKeyFile        string   // Path to TLS private-key file.
	APIKey            string   // API key required by all authenticated HTTP requests.
	HTTPPublicMethods []string // MCP methods that bypass API-key auth. Defaults to DefaultPublicMethods.
}

// CLIOverrides holds optional configuration values supplied via CLI flags.
// Every field is a raw string; conversion and validation happen in LoadConfig.
type CLIOverrides struct {
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

// LoadConfig loads configuration from environment variables, applies CLI
// overrides (which take precedence), and validates the result.
func LoadConfig(o *CLIOverrides) (*Config, error) {
	// ---- general --------------------------------------------------------
	logLevel := applyOverride(GetEnvWithDefault("LOG_LEVEL", "info"), o.LogLevel)
	logFormat := applyOverride(GetEnvWithDefault("LOG_FORMAT", "text"), o.LogFormat)
	readOnly := applyOverride(GetEnvWithDefault("READ_ONLY", "true"), o.ReadOnly)

	if !slices.Contains(logger.ValidLogLevels, logLevel) {
		fmt.Fprintf(os.Stderr, "Warning: invalid LOG_LEVEL %q, using 'info'. Valid: %v\n", logLevel, logger.ValidLogLevels)
		logLevel = "info"
	}
	if !slices.Contains(logger.ValidLogFormats, logFormat) {
		fmt.Fprintf(os.Stderr, "Warning: invalid LOG_FORMAT %q, using 'text'. Valid: %v\n", logFormat, logger.ValidLogFormats)
		logFormat = "text"
	}

	// ---- transport ------------------------------------------------------
	transport := applyOverride(GetEnvWithDefault("TRANSPORT", TransportStdio), o.Transport)
	if transport != TransportStdio && transport != TransportHTTP {
		fmt.Fprintf(os.Stderr, "Warning: invalid TRANSPORT %q, using 'stdio'.\n", transport)
		transport = TransportStdio
	}

	// ---- HTTP -----------------------------------------------------------
	httpAddr := applyOverride(GetEnvWithDefault("HTTP_ADDR", DefaultHTTPAddr), o.HTTPAddr)
	tlsEnabled := ParseBool(
		applyOverride(GetEnvWithDefault("TLS", "false"), o.TLSEnabled),
		false,
	)
	tlsCertFile := applyOverride(GetEnv("TLS_CERT_FILE"), o.TLSCertFile)
	tlsKeyFile := applyOverride(GetEnv("TLS_KEY_FILE"), o.TLSKeyFile)
	apiKey := applyOverride(GetEnv("API_KEY"), o.APIKey)

	cfg := &Config{
		ReadOnly:          ParseBool(readOnly, true),
		LogLevel:          logLevel,
		LogFormat:         logFormat,
		Transport:         transport,
		HTTPAddr:          httpAddr,
		TLSEnabled:        tlsEnabled,
		TLSCertFile:       tlsCertFile,
		TLSKeyFile:        tlsKeyFile,
		APIKey:            apiKey,
		HTTPPublicMethods: DefaultPublicMethods,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate checks for configuration errors that should abort startup.
func (c *Config) validate() error {
	if c.Transport == TransportHTTP {
		if c.TLSEnabled {
			if c.TLSCertFile == "" {
				return fmt.Errorf("TLS_CERT_FILE is required when TLS is enabled")
			}
			if c.TLSKeyFile == "" {
				return fmt.Errorf("TLS_KEY_FILE is required when TLS is enabled")
			}
		}
		if c.APIKey == "" {
			fmt.Fprintln(os.Stderr, "Warning: HTTP mode is active but no API_KEY is set — all requests will be unauthenticated")
		}
	}
	return nil
}

// applyOverride returns override if non-empty, otherwise base.
func applyOverride(base, override string) string {
	if override != "" {
		return override
	}
	return base
}

// GetEnv returns the value of an environment variable or empty string.
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvWithDefault returns the value of an environment variable or defaultValue.
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseBool parses a string to bool.
// Returns defaultValue when the string is empty or unparseable.
func ParseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: invalid boolean %q, using default %v", value, defaultValue)
		return defaultValue
	}
	return parsed
}

// ParseInt32 parses a string to int32.
// Returns defaultValue when the string is empty or unparseable.
func ParseInt32(value string, defaultValue int32) int32 {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		log.Printf("Warning: invalid integer %q, using default %v", value, defaultValue)
		return defaultValue
	}
	return int32(parsed)
}
