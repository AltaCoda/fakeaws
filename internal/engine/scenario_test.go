package engine

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestScenarioStack_FirstMatchWins(t *testing.T) {
	ss := &ScenarioStack{}

	// Push two scenarios — second pushed goes to top
	ss.Push(&Scenario{
		Name:    "catch-all",
		Matcher: func(req *ParsedRequest) bool { return true },
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			w.WriteHeader(200)
			w.Write([]byte("catch-all"))
		},
	})
	ss.Push(&Scenario{
		Name:    "specific",
		Matcher: OperationIs("SendEmail"),
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			w.WriteHeader(200)
			w.Write([]byte("specific"))
		},
	})

	req := &ParsedRequest{Operation: "SendEmail"}
	responder, id := ss.Evaluate(req)
	if responder == nil {
		t.Fatal("expected a match")
	}

	w := httptest.NewRecorder()
	responder(w, req)
	if w.Body.String() != "specific" {
		t.Fatalf("expected 'specific', got %q (matched %s)", w.Body.String(), id)
	}
}

func TestScenarioStack_OnceRemoved(t *testing.T) {
	ss := &ScenarioStack{}
	ss.Push(&Scenario{
		Name:    "once",
		Once:    true,
		Matcher: func(req *ParsedRequest) bool { return true },
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			w.WriteHeader(200)
		},
	})

	req := &ParsedRequest{}
	responder, _ := ss.Evaluate(req)
	if responder == nil {
		t.Fatal("first eval should match")
	}

	responder, _ = ss.Evaluate(req)
	if responder != nil {
		t.Fatal("second eval should not match (once scenario removed)")
	}
}

func TestScenarioStack_HitCount(t *testing.T) {
	ss := &ScenarioStack{}
	ss.Push(&Scenario{
		Name:    "counter",
		Matcher: func(req *ParsedRequest) bool { return true },
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			w.WriteHeader(200)
		},
	})

	req := &ParsedRequest{}
	for i := 0; i < 5; i++ {
		ss.Evaluate(req)
	}

	infos := ss.List()
	if len(infos) != 1 || infos[0].HitCount != 5 {
		t.Fatalf("expected hitCount=5, got %d", infos[0].HitCount)
	}
}

func TestScenarioStack_Concurrent(t *testing.T) {
	ss := &ScenarioStack{}
	ss.Push(&Scenario{
		Name:    "concurrent",
		Matcher: func(req *ParsedRequest) bool { return true },
		Responder: func(w http.ResponseWriter, req *ParsedRequest) {
			w.WriteHeader(200)
		},
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss.Evaluate(&ParsedRequest{})
		}()
	}
	wg.Wait()

	infos := ss.List()
	if infos[0].HitCount != 100 {
		t.Fatalf("expected 100, got %d", infos[0].HitCount)
	}
}

func TestMatchers_All_Any(t *testing.T) {
	req := &ParsedRequest{Service: "sesv2", Operation: "SendEmail"}

	allMatcher := All(ServiceIs("sesv2"), OperationIs("SendEmail"))
	if !allMatcher(req) {
		t.Fatal("All should match")
	}

	anyMatcher := Any(OperationIs("GetAccount"), OperationIs("SendEmail"))
	if !anyMatcher(req) {
		t.Fatal("Any should match")
	}

	noMatch := All(ServiceIs("sts"), OperationIs("SendEmail"))
	if noMatch(req) {
		t.Fatal("All should not match with wrong service")
	}
}
