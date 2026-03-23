package engine

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// Engine is the core request handling pipeline for the fake AWS server.
// It implements http.Handler.
type Engine struct {
	scenarios *ScenarioStack
	recorder  *Recorder
	router    *Router
	logger    *zap.Logger
}

// NewEngine creates an Engine with default configuration.
func NewEngine(logger *zap.Logger) *Engine {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Engine{
		scenarios: &ScenarioStack{},
		recorder:  NewRecorder(DefaultMaxRequests),
		router:    NewRouter(),
		logger:    logger.Named("engine"),
	}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read body once for both routing (STS needs body for Action) and parsing
	bodyBytes, _ := io.ReadAll(r.Body)

	// Body reader for STS routing — parses form params lazily
	bodyReader := func() map[string]any {
		m := make(map[string]any)
		ct := r.Header.Get("Content-Type")
		if ct == "application/x-www-form-urlencoded" || strings.Contains(ct, "form") {
			if vals, err := url.ParseQuery(string(bodyBytes)); err == nil {
				for k, v := range vals {
					if len(v) == 1 {
						m[k] = v[0]
					} else {
						m[k] = v
					}
				}
			}
		} else if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &m)
		}
		return m
	}

	// Route
	match := e.router.Resolve(r.Method, r.URL.Path, bodyReader)
	if match == nil {
		// Unrecognized service (S3, etc.) — return 200 OK as a permissive stub
		// so that preflight checks and other non-SES/STS calls don't fail.
		e.logger.Debug("unknown service, returning permissive stub",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		writePermissiveResponse(w, r)
		return
	}

	// Skip control plane requests — they'll be handled by a separate mux
	if match.Service == "control" {
		WriteJSONError(w, http.StatusNotFound, "NotFound", "Control plane not mounted")
		return
	}

	// Known service but unrecognized operation — 404
	if match.Operation == "" {
		e.logger.Debug("no route matched for known service",
			zap.String("service", match.Service),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		WriteError(w, match.Service, http.StatusNotFound, "UnknownOperationException",
			"No route matched: "+r.Method+" "+r.URL.Path)
		return
	}

	// Parse request
	// Restore the body so ParseRequest can read it
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	parsed := ParseRequest(r, match.Service, match.Operation)
	parsed.PathParams = match.PathParams

	// Merge path params into Body for FieldAt access
	for k, v := range match.PathParams {
		parsed.Body[k] = v
	}

	e.logger.Debug("request matched",
		zap.String("service", match.Service),
		zap.String("operation", match.Operation),
	)

	// Capture response for recording
	rec := &responseCapture{ResponseWriter: w}

	// Evaluate scenarios
	responder, scenarioID := e.scenarios.Evaluate(parsed)
	if responder == nil {
		responder = DefaultHandler(match.Operation)
	}

	// Execute responder
	responder(rec, parsed)

	// Record
	e.recorder.Append(RecordedRequest{
		Timestamp:       parsed.Timestamp,
		Service:         parsed.Service,
		Operation:       parsed.Operation,
		Method:          parsed.Method,
		Path:            parsed.Path,
		Headers:         parsed.Headers,
		Body:            parsed.Body,
		Region:          parsed.Region,
		AccessKeyID:     parsed.AccessKeyID,
		MatchedScenario: scenarioID,
		ResponseCode:    rec.statusCode,
		ResponseBody:    rec.body.String(),
	})
}

// AddScenario pushes a scenario onto the stack.
func (e *Engine) AddScenario(s *Scenario) string {
	return e.scenarios.Push(s)
}

// RemoveScenario removes a scenario by ID.
func (e *Engine) RemoveScenario(id string) bool {
	return e.scenarios.Remove(id)
}

// ListScenarios returns info for all active scenarios.
func (e *Engine) ListScenarios() []ScenarioInfo {
	return e.scenarios.List()
}

// ClearScenarios removes all scenarios.
func (e *Engine) ClearScenarios() {
	e.scenarios.Clear()
}

// Requests returns all recorded requests.
func (e *Engine) Requests() []RecordedRequest {
	return e.recorder.All()
}

// RequestsFor returns recorded requests matching service and operation.
func (e *Engine) RequestsFor(service, operation string) []RecordedRequest {
	return e.recorder.For(service, operation)
}

// ClearRequests removes all recorded requests.
func (e *Engine) ClearRequests() {
	e.recorder.Clear()
}

// CountRequests returns the number of requests matching service and operation.
func (e *Engine) CountRequests(service, operation string) int {
	return e.recorder.Count(service, operation)
}

// LastRequest returns the most recent request matching service and operation.
func (e *Engine) LastRequest(service, operation string) *RecordedRequest {
	return e.recorder.Last(service, operation)
}

// Reset clears all scenarios and recorded requests.
func (e *Engine) Reset() {
	e.ClearScenarios()
	e.ClearRequests()
}
