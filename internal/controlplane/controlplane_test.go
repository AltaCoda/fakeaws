package controlplane_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/altacoda/fakeaws/internal/controlplane"
	"github.com/altacoda/fakeaws/internal/engine"
	"go.uber.org/zap"
)

func setup() (*engine.Engine, *httptest.Server) {
	e := engine.NewEngine(zap.NewNop())
	cp := controlplane.New(e, zap.NewNop())
	mux := http.NewServeMux()
	mux.Handle("/_control/", cp.Handler())
	mux.Handle("/", e)
	return e, httptest.NewServer(mux)
}

func get(srv *httptest.Server, path string) (*http.Response, map[string]any) {
	resp, _ := http.Get(srv.URL + path)
	return resp, readJSON(resp)
}

func post(srv *httptest.Server, path string, body any) (*http.Response, map[string]any) {
	b, _ := json.Marshal(body)
	resp, _ := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	return resp, readJSON(resp)
}

func del(srv *httptest.Server, path string) (*http.Response, map[string]any) {
	req, _ := http.NewRequest("DELETE", srv.URL+path, nil)
	resp, _ := http.DefaultClient.Do(req)
	return resp, readJSON(resp)
}

func readJSON(resp *http.Response) map[string]any {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}

func TestScenarioCRUD(t *testing.T) {
	_, srv := setup()
	defer srv.Close()

	// List — empty
	resp, data := get(srv, "/_control/scenarios")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Create
	resp, data = post(srv, "/_control/scenarios", map[string]any{
		"name":      "test-throttle",
		"operation": "SendEmail",
		"response":  map[string]any{"type": "throttle"},
	})
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	id := data["id"].(string)
	if id == "" {
		t.Fatal("expected scenario ID")
	}

	// List — has one
	resp, _ = get(srv, "/_control/scenarios")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Delete
	resp, _ = del(srv, "/_control/scenarios/"+id)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Delete nonexistent
	resp, _ = del(srv, "/_control/scenarios/nonexistent")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestScenarioFromJSON_Error(t *testing.T) {
	_, srv := setup()
	defer srv.Close()

	// Missing error_code
	resp, data := post(srv, "/_control/scenarios", map[string]any{
		"operation": "SendEmail",
		"response":  map[string]any{"type": "error"},
	})
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if data["error"] == nil {
		t.Fatal("expected error message")
	}
}

func TestRequestLog(t *testing.T) {
	e, srv := setup()
	defer srv.Close()

	// Send a request through the AWS engine
	http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", bytes.NewReader([]byte(`{}`)))

	// List requests
	resp, data := get(srv, "/_control/requests")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	total := data["total"].(float64)
	if total != 1 {
		t.Fatalf("expected 1 request, got %v", total)
	}

	// Filter by operation
	resp, data = get(srv, "/_control/requests?operation=SendEmail")
	if data["total"].(float64) != 1 {
		t.Fatalf("expected 1 filtered request")
	}

	resp, data = get(srv, "/_control/requests?operation=GetAccount")
	if data["total"].(float64) != 0 {
		t.Fatalf("expected 0 filtered requests")
	}

	// Get detail
	reqs := e.Requests()
	resp, _ = get(srv, "/_control/requests/"+reqs[0].ID)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Clear
	resp, _ = del(srv, "/_control/requests")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	resp, data = get(srv, "/_control/requests")
	if data["total"].(float64) != 0 {
		t.Fatalf("expected 0 after clear")
	}
}

func TestReset(t *testing.T) {
	_, srv := setup()
	defer srv.Close()

	// Add a scenario
	post(srv, "/_control/scenarios", map[string]any{
		"operation": "SendEmail",
		"response":  map[string]any{"type": "throttle"},
	})

	// Send a request
	http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", bytes.NewReader([]byte(`{}`)))

	// Reset
	resp, _ := post(srv, "/_control/reset", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify both cleared
	_, data := get(srv, "/_control/requests")
	if data["total"].(float64) != 0 {
		t.Fatal("expected 0 requests after reset")
	}
}

func TestPresets(t *testing.T) {
	_, srv := setup()
	defer srv.Close()

	// List presets
	resp, _ := http.Get(srv.URL + "/_control/presets")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Apply throttle preset
	resp, _ = post(srv, "/_control/scenarios/presets/throttle_all_sends", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Send email — should be throttled
	resp, _ = http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", bytes.NewReader([]byte(`{}`)))
	if resp.StatusCode != 429 {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}

	// Apply happy_path — clears throttle
	post(srv, "/_control/scenarios/presets/happy_path", nil)

	resp, _ = http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", bytes.NewReader([]byte(`{}`)))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 after happy_path, got %d", resp.StatusCode)
	}
}

func TestPreset_SandboxMode_RequiresConfig(t *testing.T) {
	_, srv := setup()
	defer srv.Close()

	resp, data := post(srv, "/_control/scenarios/presets/sandbox_mode", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if data["error"] == nil {
		t.Fatal("expected error about verified_addresses")
	}
}

func TestDashboard(t *testing.T) {
	_, srv := setup()
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/_control/dashboard")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html, got %s", ct)
	}
}
