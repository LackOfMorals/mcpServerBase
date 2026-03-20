package unit

import (
	"encoding/json"
	"testing"

	"github.com/LackOfMorals/mcpServerBase/internal/server"
)

// ---- ToolDef JSON serialisation -----------------------------------------

func TestToolDef_Serialisation_HandlerExcluded(t *testing.T) {
	td := &server.ToolDef{
		ID:       "ser-test",
		Name:     "Serialisation Test",
		Type:     server.ToolTypeCreate,
		ReadOnly: false,
		Parameters: []server.ToolParam{
			{Name: "input", Type: "string", Required: true},
		},
		Handler: echoHandler, // must NOT appear in JSON
	}

	data, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := m["handler"]; ok {
		t.Error("handler field must not appear in serialised ToolDef")
	}
	if m["id"] != "ser-test" {
		t.Errorf("expected id='ser-test', got %v", m["id"])
	}
}

func TestToolDef_Serialisation_ParametersIncluded(t *testing.T) {
	td := &server.ToolDef{
		ID:   "with-params",
		Type: server.ToolTypeRead,
		Parameters: []server.ToolParam{
			{Name: "limit", Type: "integer", Required: false, Default: 10},
		},
	}

	data, _ := json.Marshal(td)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck

	params, ok := m["parameters"].([]interface{})
	if !ok || len(params) != 1 {
		t.Fatalf("expected 1 parameter, got %v", m["parameters"])
	}
	p := params[0].(map[string]interface{})
	if p["name"] != "limit" {
		t.Errorf("expected name='limit', got %v", p["name"])
	}
}

func TestToolDef_Serialisation_NoParametersOmitted(t *testing.T) {
	td := &server.ToolDef{ID: "no-params", Type: server.ToolTypeRead}
	data, _ := json.Marshal(td)

	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck
	if _, ok := m["parameters"]; ok {
		t.Error("parameters should be omitted when nil/empty")
	}
}

// ---- ToolSummary --------------------------------------------------------

func TestToolSummary_AllFieldsPresent(t *testing.T) {
	s := server.ToolSummary{
		ID:       "summ-1",
		Name:     "Summary One",
		Type:     server.ToolTypeDelete,
		ReadOnly: false,
	}
	data, _ := json.Marshal(s)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck

	for _, key := range []string{"id", "name", "type", "readonly"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected field %q in ToolSummary JSON", key)
		}
	}
}

// ---- ToolType constants -------------------------------------------------

func TestToolType_Values(t *testing.T) {
	cases := []struct {
		tt   server.ToolType
		want string
	}{
		{server.ToolTypeList, "list"},
		{server.ToolTypeRead, "read"},
		{server.ToolTypeCreate, "create"},
		{server.ToolTypeUpdate, "update"},
		{server.ToolTypeDelete, "delete"},
	}
	for _, tc := range cases {
		if string(tc.tt) != tc.want {
			t.Errorf("ToolType %v: expected %q, got %q", tc.tt, tc.want, string(tc.tt))
		}
	}
}

// ---- JobStatus constants ------------------------------------------------

func TestJobStatus_Values(t *testing.T) {
	cases := []struct {
		js   server.JobStatus
		want string
	}{
		{server.JobStatusPending, "pending"},
		{server.JobStatusRunning, "running"},
		{server.JobStatusCompleted, "completed"},
		{server.JobStatusFailed, "failed"},
	}
	for _, tc := range cases {
		if string(tc.js) != tc.want {
			t.Errorf("JobStatus %v: expected %q, got %q", tc.js, tc.want, string(tc.js))
		}
	}
}

// ---- ToolParam default field omitted when zero -------------------------

func TestToolParam_DefaultOmittedWhenNil(t *testing.T) {
	p := server.ToolParam{Name: "x", Type: "string", Required: true}
	data, _ := json.Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck
	if _, ok := m["default"]; ok {
		t.Error("default should be omitted when not set")
	}
}

func TestToolParam_DefaultIncludedWhenSet(t *testing.T) {
	p := server.ToolParam{Name: "limit", Type: "integer", Default: float64(25)}
	data, _ := json.Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m) //nolint:errcheck
	if m["default"] != float64(25) {
		t.Errorf("expected default=25, got %v", m["default"])
	}
}
