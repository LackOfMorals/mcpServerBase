package httpsvr

// White-box unit tests for middleware.go.
// Being in the same package lets us call checkAPIKey and buildPublicMethodSet
// directly without exporting them.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- checkAPIKey ---------------------------------------------------------

func TestCheckAPIKey_BearerMatch(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer secret-key")
	if !checkAPIKey(r, "secret-key") {
		t.Error("expected true for matching Bearer token")
	}
}

func TestCheckAPIKey_XAPIKeyMatch(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("X-API-Key", "secret-key")
	if !checkAPIKey(r, "secret-key") {
		t.Error("expected true for matching X-API-Key header")
	}
}

func TestCheckAPIKey_WrongBearer(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer wrong")
	if checkAPIKey(r, "secret-key") {
		t.Error("expected false for wrong Bearer token")
	}
}

func TestCheckAPIKey_WrongXAPIKey(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("X-API-Key", "wrong")
	if checkAPIKey(r, "secret-key") {
		t.Error("expected false for wrong X-API-Key")
	}
}

func TestCheckAPIKey_NoHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	if checkAPIKey(r, "secret-key") {
		t.Error("expected false when no auth headers are present")
	}
}

func TestCheckAPIKey_BearerPrefixOnly(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer ") // empty token after prefix
	if checkAPIKey(r, "secret-key") {
		t.Error("expected false for empty bearer token")
	}
}

func TestCheckAPIKey_NonBearerScheme(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	if checkAPIKey(r, "secret-key") {
		t.Error("expected false for non-Bearer Authorization scheme")
	}
}

// ---- buildPublicMethodSet ------------------------------------------------

func TestBuildPublicMethodSet_Membership(t *testing.T) {
	methods := []string{"initialize", "ping", "tools/list"}
	set := buildPublicMethodSet(methods)

	for _, m := range methods {
		if _, ok := set[m]; !ok {
			t.Errorf("expected %q to be in set", m)
		}
	}
}

func TestBuildPublicMethodSet_NonMembership(t *testing.T) {
	set := buildPublicMethodSet([]string{"initialize"})
	if _, ok := set["tools/call"]; ok {
		t.Error("expected 'tools/call' not to be in set")
	}
}

func TestBuildPublicMethodSet_Empty(t *testing.T) {
	set := buildPublicMethodSet(nil)
	if len(set) != 0 {
		t.Errorf("expected empty set, got %d entries", len(set))
	}
}

// ---- apiKeyAuthMiddleware ------------------------------------------------

// downstream is a simple handler that records whether it was called.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func postWithMethod(method string) *http.Request {
	body, _ := json.Marshal(map[string]string{"jsonrpc": "2.0", "method": method})
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestAPIKeyAuth_NoKeyConfigured_AllRequestsPass(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "", buildPublicMethodSet(nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if !called {
		t.Error("expected downstream to be called when no API key is configured")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPIKeyAuth_PublicMethod_NoKeyRequired(t *testing.T) {
	called := false
	publicMethods := buildPublicMethodSet([]string{"initialize", "ping"})
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", publicMethods)

	for _, method := range []string{"initialize", "ping"} {
		called = false
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, postWithMethod(method))

		if !called {
			t.Errorf("expected downstream to be called for public method %q", method)
		}
		if rr.Code != http.StatusOK {
			t.Errorf("public method %q: expected 200, got %d", method, rr.Code)
		}
	}
}

func TestAPIKeyAuth_ProtectedMethod_ValidBearerToken(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", buildPublicMethodSet(nil))

	req := postWithMethod("tools/call")
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected downstream to be called with valid Bearer token")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPIKeyAuth_ProtectedMethod_ValidXAPIKey(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", buildPublicMethodSet(nil))

	req := postWithMethod("tools/call")
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected downstream to be called with valid X-API-Key")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPIKeyAuth_ProtectedMethod_NoKey_Returns401(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", buildPublicMethodSet(nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, postWithMethod("tools/call"))

	if called {
		t.Error("expected downstream NOT to be called without a key")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAPIKeyAuth_ProtectedMethod_WrongKey_Returns401(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", buildPublicMethodSet(nil))

	req := postWithMethod("tools/call")
	req.Header.Set("Authorization", "Bearer wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected downstream NOT to be called with wrong key")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAPIKeyAuth_GET_ValidKey_Passes(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", buildPublicMethodSet(nil))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected downstream to be called for GET with valid key")
	}
}

func TestAPIKeyAuth_GET_NoKey_Returns401(t *testing.T) {
	called := false
	handler := apiKeyAuthMiddleware(okHandler(&called), "secret", buildPublicMethodSet(nil))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected downstream NOT to be called for unauthenticated GET")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAPIKeyAuth_BodyRestoredAfterPeek(t *testing.T) {
	// Verify the body is readable by the downstream handler after the middleware
	// has peeked at it to determine the JSON-RPC method.
	var receivedBody []byte
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	handler := apiKeyAuthMiddleware(downstream, "secret", buildPublicMethodSet(nil))

	req := postWithMethod("tools/call")
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(receivedBody) == 0 {
		t.Error("expected downstream to receive the request body after middleware peek")
	}
	var env jsonRPCEnvelope
	if err := json.Unmarshal(receivedBody, &env); err != nil {
		t.Errorf("downstream body is not valid JSON: %v", err)
	}
	if env.Method != "tools/call" {
		t.Errorf("expected method='tools/call', got %q", env.Method)
	}
}

// ---- loggingMiddleware ---------------------------------------------------

func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	cases := []struct {
		writeStatus  int
		expectedCode int
	}{
		{http.StatusOK, http.StatusOK},
		{http.StatusUnauthorized, http.StatusUnauthorized},
		{http.StatusNotFound, http.StatusNotFound},
		{http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tc := range cases {
		downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.writeStatus)
		})
		handler := loggingMiddleware(downstream)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/mcp", nil))

		if rr.Code != tc.expectedCode {
			t.Errorf("writeStatus=%d: expected recorder code %d, got %d",
				tc.writeStatus, tc.expectedCode, rr.Code)
		}
	}
}

func TestLoggingMiddleware_DefaultStatus200WhenNotExplicitlySet(t *testing.T) {
	// If the handler writes a body without calling WriteHeader, the status
	// should default to 200 in our responseWriter.
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok")) //nolint:errcheck
	})
	handler := loggingMiddleware(downstream)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected default 200, got %d", rr.Code)
	}
}

func TestLoggingMiddleware_CallsDownstream(t *testing.T) {
	called := false
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	loggingMiddleware(downstream).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/mcp", nil),
	)
	if !called {
		t.Error("expected logging middleware to call the downstream handler")
	}
}

// ---- responseWriter ------------------------------------------------------

func TestResponseWriter_WriteHeaderOnce(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := newResponseWriter(rr)

	rw.WriteHeader(http.StatusTeapot)
	rw.WriteHeader(http.StatusOK) // second call must be ignored

	if rw.statusCode != http.StatusTeapot {
		t.Errorf("expected statusCode=%d, got %d", http.StatusTeapot, rw.statusCode)
	}
}
