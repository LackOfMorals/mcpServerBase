package config

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/LackOfMorals/mcpServerBase/internal/logger"
)

// Config holds the application configuration
type Config struct {
	ReadOnly  bool   // Disables tools that would make changes.  True by default
	LogLevel  string // Logging level to use.  Default  Info
	LogFormat string //  Log format to use. Default Text
}

// CLIOverrides holds optional configuration values from CLI flags
type CLIOverrides struct {
	ReadOnly  string
	LogLevel  string
	LogFormat string
}

// LoadConfig loads configuration from environment variables, applies CLI overrides, and validates.
// CLI flag values take precedence over environment variables.
// Returns an error if required configuration is missing or invalid.
func LoadConfig(cliOverrides *CLIOverrides) (*Config, error) {
	// Required

	// Optional with defaults if not set
	logLevel := GetEnvWithDefault("LOG_LEVEL", "info")
	logFormat := GetEnvWithDefault("LOG_FORMAT", "text")
	readOnly := GetEnvWithDefault("READ_ONLY", "true")

	// Validate log level and use default if invalid
	if !slices.Contains(logger.ValidLogLevels, logLevel) {
		fmt.Fprintf(os.Stderr, "Warning: invalid LOG_LEVEL '%s', using default 'info'. Valid values: %v\n", logLevel, logger.ValidLogLevels)
		logLevel = "info"
	}

	// Validate log format and use default if invalid
	if !slices.Contains(logger.ValidLogFormats, logFormat) {
		fmt.Fprintf(os.Stderr, "Warning: invalid LOG_FORMAT '%s', using default 'text'. Valid values: %v\n", logFormat, logger.ValidLogFormats)
		logFormat = "text"
	}

	cfg := &Config{
		ReadOnly:  ParseBool(readOnly, true),
		LogLevel:  logLevel,
		LogFormat: logFormat,
	}

	return cfg, nil
}

// GetEnv returns the value of an environment variable or empty string if not set
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvWithDefault returns the value of an environment variable or a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseBool parses a string to bool using strconv.ParseBool.
// Returns the default value if the string is empty or invalid.
// Logs a warning if the value is non-empty but invalid.
// Accepts: "1", "t", "T", "true", "True", "TRUE" for true
//
//	"0", "f", "F", "false", "False", "FALSE" for false
func ParseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: Invalid boolean value %q, using default: %v", value, defaultValue)
		return defaultValue
	}
	return parsed
}

// ParseInt32 parses a string to int32.
// Returns the default value if the string is empty or invalid.
func ParseInt32(value string, defaultValue int32) int32 {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		log.Printf("Warning: Invalid integer value %q, using default: %v", value, defaultValue)
		return defaultValue
	}
	return int32(parsed)
}
