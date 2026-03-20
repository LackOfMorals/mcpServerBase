package unit

import (
	"os"
	"testing"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
)

// ---- transport selection ------------------------------------------------

func TestConfig_DefaultTransportIsStdio(t *testing.T) {
	cfg, err := config.LoadConfig(&config.CLIOverrides{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != config.TransportStdio {
		t.Errorf("expected default transport 'stdio', got %q", cfg.Transport)
	}
}

func TestConfig_TransportHTTP_ViaCLI(t *testing.T) {
	cfg, err := config.LoadConfig(&config.CLIOverrides{Transport: "http"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != config.TransportHTTP {
		t.Errorf("expected transport 'http', got %q", cfg.Transport)
	}
}

func TestConfig_InvalidTransport_FallsBackToStdio(t *testing.T) {
	cfg, err := config.LoadConfig(&config.CLIOverrides{Transport: "grpc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != config.TransportStdio {
		t.Errorf("invalid transport should fall back to 'stdio', got %q", cfg.Transport)
	}
}

func TestConfig_TransportHTTP_ViaEnv(t *testing.T) {
	t.Setenv("TRANSPORT", "http")
	cfg, err := config.LoadConfig(&config.CLIOverrides{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Transport != config.TransportHTTP {
		t.Errorf("expected 'http' from env, got %q", cfg.Transport)
	}
}

// ---- HTTP address -------------------------------------------------------

func TestConfig_DefaultHTTPAddr(t *testing.T) {
	cfg, _ := config.LoadConfig(&config.CLIOverrides{Transport: "http"})
	if cfg.HTTPAddr != config.DefaultHTTPAddr {
		t.Errorf("expected default addr %q, got %q", config.DefaultHTTPAddr, cfg.HTTPAddr)
	}
}

func TestConfig_CustomHTTPAddr_ViaCLI(t *testing.T) {
	cfg, _ := config.LoadConfig(&config.CLIOverrides{
		Transport: "http",
		HTTPAddr:  "0.0.0.0:9090",
	})
	if cfg.HTTPAddr != "0.0.0.0:9090" {
		t.Errorf("expected '0.0.0.0:9090', got %q", cfg.HTTPAddr)
	}
}

func TestConfig_CustomHTTPAddr_ViaEnv(t *testing.T) {
	t.Setenv("HTTP_ADDR", "0.0.0.0:7777")
	cfg, _ := config.LoadConfig(&config.CLIOverrides{Transport: "http"})
	if cfg.HTTPAddr != "0.0.0.0:7777" {
		t.Errorf("expected '0.0.0.0:7777', got %q", cfg.HTTPAddr)
	}
}

// ---- TLS validation -----------------------------------------------------

func TestConfig_TLS_MissingCert_ReturnsError(t *testing.T) {
	_, err := config.LoadConfig(&config.CLIOverrides{
		Transport:  "http",
		TLSEnabled: "true",
		TLSKeyFile: "/some/key.pem",
		// TLSCertFile intentionally absent
	})
	if err == nil {
		t.Error("expected error when TLS cert file is missing")
	}
}

func TestConfig_TLS_MissingKey_ReturnsError(t *testing.T) {
	_, err := config.LoadConfig(&config.CLIOverrides{
		Transport:   "http",
		TLSEnabled:  "true",
		TLSCertFile: "/some/cert.pem",
		// TLSKeyFile intentionally absent
	})
	if err == nil {
		t.Error("expected error when TLS key file is missing")
	}
}

func TestConfig_TLS_BothFiles_NoError(t *testing.T) {
	cfg, err := config.LoadConfig(&config.CLIOverrides{
		Transport:   "http",
		TLSEnabled:  "true",
		TLSCertFile: "/some/cert.pem",
		TLSKeyFile:  "/some/key.pem",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.TLSEnabled {
		t.Error("expected TLSEnabled=true")
	}
	if cfg.TLSCertFile != "/some/cert.pem" {
		t.Errorf("expected cert '/some/cert.pem', got %q", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "/some/key.pem" {
		t.Errorf("expected key '/some/key.pem', got %q", cfg.TLSKeyFile)
	}
}

func TestConfig_TLS_DisabledByDefault(t *testing.T) {
	cfg, _ := config.LoadConfig(&config.CLIOverrides{Transport: "http"})
	if cfg.TLSEnabled {
		t.Error("expected TLS disabled by default")
	}
}

func TestConfig_TLS_NotValidatedInStdioMode(t *testing.T) {
	// TLS settings are only validated in HTTP mode; in stdio mode they should
	// never cause an error even if TLS=true but cert/key are missing.
	_, err := config.LoadConfig(&config.CLIOverrides{
		Transport:  "stdio",
		TLSEnabled: "true",
		// No cert or key — should not error in stdio mode
	})
	if err != nil {
		t.Errorf("TLS validation should be skipped in stdio mode, got error: %v", err)
	}
}

// ---- API key ------------------------------------------------------------

func TestConfig_APIKey_ViaCLI(t *testing.T) {
	cfg, _ := config.LoadConfig(&config.CLIOverrides{
		Transport: "http",
		APIKey:    "my-key",
	})
	if cfg.APIKey != "my-key" {
		t.Errorf("expected APIKey='my-key', got %q", cfg.APIKey)
	}
}

func TestConfig_APIKey_ViaEnv(t *testing.T) {
	t.Setenv("API_KEY", "env-key")
	cfg, _ := config.LoadConfig(&config.CLIOverrides{Transport: "http"})
	if cfg.APIKey != "env-key" {
		t.Errorf("expected APIKey='env-key', got %q", cfg.APIKey)
	}
}

func TestConfig_CLIOverrides_APIKey_TakesPrecedenceOverEnv(t *testing.T) {
	t.Setenv("API_KEY", "env-key")
	cfg, _ := config.LoadConfig(&config.CLIOverrides{
		Transport: "http",
		APIKey:    "cli-key",
	})
	if cfg.APIKey != "cli-key" {
		t.Errorf("CLI override should take precedence: expected 'cli-key', got %q", cfg.APIKey)
	}
}

// ---- public methods defaults --------------------------------------------

func TestConfig_DefaultPublicMethodsPopulated(t *testing.T) {
	cfg, _ := config.LoadConfig(&config.CLIOverrides{Transport: "http"})

	expected := map[string]bool{
		"initialize":               true,
		"notifications/initialized": true,
		"ping":                     true,
		"tools/list":               true,
	}
	for _, m := range cfg.HTTPPublicMethods {
		delete(expected, m)
	}
	for missing := range expected {
		t.Errorf("expected default public method %q to be present", missing)
	}
}

// ---- CLI overrides take precedence over env vars ------------------------

func TestConfig_CLIOverride_LogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "error")
	cfg, _ := config.LoadConfig(&config.CLIOverrides{LogLevel: "debug"})
	if cfg.LogLevel != "debug" {
		t.Errorf("CLI --log-level should override env: expected 'debug', got %q", cfg.LogLevel)
	}
}

func TestConfig_CLIOverride_ReadOnly(t *testing.T) {
	t.Setenv("READ_ONLY", "true")
	cfg, _ := config.LoadConfig(&config.CLIOverrides{ReadOnly: "false"})
	if cfg.ReadOnly {
		t.Error("CLI --read-only=false should override env READ_ONLY=true")
	}
}

func TestConfig_EnvVar_UsedWhenNoCLIOverride(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warning")
	cfg, _ := config.LoadConfig(&config.CLIOverrides{})
	if cfg.LogLevel != "warning" {
		t.Errorf("expected log level 'warning' from env, got %q", cfg.LogLevel)
	}
}

// ---- env cleanup guard --------------------------------------------------
// Ensure we haven't leaked anything from a failed test by verifying
// t.Setenv properly restores values.

func TestConfig_EnvCleanupVerification(t *testing.T) {
	if os.Getenv("TRANSPORT") != "" {
		t.Skip("TRANSPORT env var is set externally; skipping cleanup verification")
	}
	cfg, _ := config.LoadConfig(&config.CLIOverrides{})
	if cfg.Transport != config.TransportStdio {
		t.Errorf("expected default stdio transport, got %q — possible env leak", cfg.Transport)
	}
}
