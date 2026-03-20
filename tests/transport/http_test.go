// Package transport contains integration tests for the HTTP transport layer.
//
// Tests use httptest.NewServer to spin up a real listener, drive requests
// through the full middleware chain (logging → auth → mock MCP handler),
// and assert on HTTP response codes and body content.
//
// Run with:
//
//	go test ./tests/transport/ -v
//	go test -race ./tests/transport/ -v
package transport

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LackOfMorals/mcpServerBase/internal/config"
	httpsvr "github.com/LackOfMorals/mcpServerBase/internal/transport/http"
)

// ---- helpers -------------------------------------------------------------

// echoBackend is a minimal MCP-like handler that returns 200 + the request
// body as JSON so tests can verify the body was forwarded intact.
var echoBackend = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body) //nolint:errcheck
})

// rpcBody returns a JSON-RPC POST body for the given method.
func rpcBody(method string) io.Reader {
	b, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	})
	return bytes.NewReader(b)
}

// newTestServer builds a middleware chain from cfg and the given backend, and
// wraps it in an httptest.Server.  Caller is responsible for closing.
func newTestServer(t *testing.T, cfg *config.Config, backend http.Handler) *httptest.Server {
	t.Helper()
	handler := httpsvr.NewHandler(cfg.APIKey, cfg.HTTPPublicMethods, backend)
	return httptest.NewServer(handler)
}

// post sends a POST request to url with a JSON-RPC body and optional headers.
func post(t *testing.T, url, method string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/mcp", rpcBody(method))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

// get sends a GET request to url/mcp with optional headers.
func get(t *testing.T, url string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url+"/mcp", nil)
	if err != nil {
		t.Fatalf("build GET request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

// ---- no API key (open server) -------------------------------------------

func TestHTTP_NoAPIKey_AllRequestsPass(t *testing.T) {
	cfg := &config.Config{HTTPPublicMethods: config.DefaultPublicMethods}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	for _, method := range []string{"initialize", "tools/call", "tools/list", "ping"} {
		resp := post(t, srv.URL, method, nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("no-key server, method=%q: expected 200, got %d", method, resp.StatusCode)
		}
	}
}

// ---- public methods bypass auth -----------------------------------------

func TestHTTP_PublicMethods_NoAuthRequired(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods, // initialize, notifications/initialized, ping, tools/list
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	for _, method := range config.DefaultPublicMethods {
		resp := post(t, srv.URL, method, nil) // no auth header
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("public method %q without auth: expected 200, got %d", method, resp.StatusCode)
		}
	}
}

// ---- protected methods require auth -------------------------------------

func TestHTTP_ProtectedMethod_NoKey_Returns401(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := post(t, srv.URL, "tools/call", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHTTP_ProtectedMethod_WrongKey_Returns401(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := post(t, srv.URL, "tools/call", map[string]string{
		"Authorization": "Bearer wrong-key",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHTTP_ProtectedMethod_BearerToken_Returns200(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := post(t, srv.URL, "tools/call", map[string]string{
		"Authorization": "Bearer secret",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHTTP_ProtectedMethod_XAPIKey_Returns200(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := post(t, srv.URL, "tools/call", map[string]string{
		"X-API-Key": "secret",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ---- GET requests -------------------------------------------------------

func TestHTTP_GET_WithValidKey_Passes(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := get(t, srv.URL, map[string]string{"X-API-Key": "secret"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHTTP_GET_NoKey_Returns401(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := get(t, srv.URL, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// ---- body is forwarded intact -------------------------------------------

func TestHTTP_BodyForwardedToBackend(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: config.DefaultPublicMethods,
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	resp := post(t, srv.URL, "tools/call", map[string]string{
		"X-API-Key": "secret",
	})
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "tools/call") {
		t.Errorf("expected backend to receive original body, got: %s", body)
	}
}

// ---- custom public method list ------------------------------------------

func TestHTTP_CustomPublicMethods_OnlyListedBypass(t *testing.T) {
	cfg := &config.Config{
		APIKey:            "secret",
		HTTPPublicMethods: []string{"initialize"}, // only initialize is public
	}
	srv := newTestServer(t, cfg, echoBackend)
	defer srv.Close()

	// initialize should pass without key.
	resp := post(t, srv.URL, "initialize", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("initialize without key: expected 200, got %d", resp.StatusCode)
	}

	// ping is NOT in the custom list — should require auth.
	resp = post(t, srv.URL, "ping", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("ping without key (not in public list): expected 401, got %d", resp.StatusCode)
	}
}
