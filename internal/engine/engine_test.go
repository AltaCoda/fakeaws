package engine

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func newTestEngine() *Engine {
	return NewEngine(zap.NewNop())
}

func TestEngine_DefaultSendEmail(t *testing.T) {
	e := newTestEngine()
	srv := httptest.NewServer(e)
	defer srv.Close()

	body := `{"Destination":{"ToAddresses":["test@example.com"]},"Content":{"Simple":{"Subject":{"Data":"Hi"},"Body":{"Text":{"Data":"Hello"}}}}}`
	resp, err := http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(respBody), "MessageId") {
		t.Fatalf("expected MessageId in response, got %s", respBody)
	}

	// Verify recording
	reqs := e.RequestsFor("sesv2", "SendEmail")
	if len(reqs) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(reqs))
	}
}

func TestEngine_ScenarioOverridesDefault(t *testing.T) {
	e := newTestEngine()
	srv := httptest.NewServer(e)
	defer srv.Close()

	e.AddScenario(&Scenario{
		Name:    "bounce",
		Matcher: OperationIs("SendEmail"),
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			WriteJSONError(w, 400, "MessageRejected", "Email address is on bounce list")
		},
	})

	resp, err := http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestEngine_UnknownRoute_404(t *testing.T) {
	e := newTestEngine()
	srv := httptest.NewServer(e)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v2/email/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestEngine_STS_GetCallerIdentity(t *testing.T) {
	e := newTestEngine()
	srv := httptest.NewServer(e)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/", "application/x-www-form-urlencoded", strings.NewReader("Action=GetCallerIdentity&Version=2011-06-15"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "GetCallerIdentityResult") {
		t.Fatalf("expected GetCallerIdentityResult, got %s", body)
	}
}

func TestEngine_Reset(t *testing.T) {
	e := newTestEngine()

	e.AddScenario(&Scenario{
		Name:    "test",
		Matcher: func(req *ParsedRequest) bool { return true },
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			w.WriteHeader(200)
		},
	})

	// Generate a recorded request via actual HTTP call
	srv := httptest.NewServer(e)
	defer srv.Close()
	http.Post(srv.URL+"/v2/email/outbound-emails", "application/json", strings.NewReader(`{}`))

	e.Reset()

	if len(e.ListScenarios()) != 0 {
		t.Fatal("expected 0 scenarios after reset")
	}
	if len(e.Requests()) != 0 {
		t.Fatal("expected 0 requests after reset")
	}
}
