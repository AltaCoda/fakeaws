package controlplane

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/altacoda/fakeaws/internal/engine"
	"go.uber.org/zap"
)

// ControlPlane provides HTTP endpoints for managing the fake server.
type ControlPlane struct {
	engine  *engine.Engine
	presets map[string]PresetFunc
	logger  *zap.Logger
}

// New creates a ControlPlane wrapping the given engine.
func New(e *engine.Engine, logger *zap.Logger) *ControlPlane {
	if logger == nil {
		logger = zap.NewNop()
	}
	cp := &ControlPlane{
		engine: e,
		logger: logger.Named("controlplane"),
	}
	cp.presets = builtinPresets()
	return cp
}

// Handler returns an http.Handler for all /_control/ routes.
func (cp *ControlPlane) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /_control/scenarios", cp.listScenarios)
	mux.HandleFunc("POST /_control/scenarios", cp.createScenario)
	mux.HandleFunc("DELETE /_control/scenarios/{id}", cp.deleteScenario)

	mux.HandleFunc("GET /_control/requests", cp.listRequests)
	mux.HandleFunc("GET /_control/requests/{id}", cp.getRequest)
	mux.HandleFunc("DELETE /_control/requests", cp.clearRequests)

	mux.HandleFunc("POST /_control/scenarios/presets/{name}", cp.applyPreset)
	mux.HandleFunc("GET /_control/presets", cp.listPresets)

	mux.HandleFunc("POST /_control/reset", cp.reset)

	mux.HandleFunc("GET /_control/dashboard", cp.serveDashboard)
	mux.HandleFunc("GET /_control/dashboard/", cp.serveDashboard)

	// Wrap with CORS
	return cp.corsMiddleware(mux)
}

func (cp *ControlPlane) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Scenario CRUD ---

func (cp *ControlPlane) listScenarios(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, cp.engine.ListScenarios())
}

func (cp *ControlPlane) createScenario(w http.ResponseWriter, r *http.Request) {
	var input ScenarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	scenario, err := ScenarioFromJSON(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	id := cp.engine.AddScenario(scenario)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (cp *ControlPlane) deleteScenario(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if cp.engine.RemoveScenario(id) {
		writeJSON(w, http.StatusOK, map[string]string{"removed": id})
	} else {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "scenario not found: " + id})
	}
}

// --- Request Log ---

func (cp *ControlPlane) listRequests(w http.ResponseWriter, r *http.Request) {
	reqs := cp.engine.Requests()

	// Apply filters
	if op := r.URL.Query().Get("operation"); op != "" {
		reqs = filterByOperation(reqs, op)
	}
	if svc := r.URL.Query().Get("service"); svc != "" {
		reqs = filterByService(reqs, svc)
	}
	if since := r.URL.Query().Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			reqs = filterSince(reqs, t)
		}
	}

	total := len(reqs)

	if last := r.URL.Query().Get("last"); last != "" {
		if n := parseInt(last); n > 0 && n < len(reqs) {
			reqs = reqs[len(reqs)-n:]
		}
	}

	// Strip headers from list view for size
	slim := make([]map[string]any, len(reqs))
	for i, req := range reqs {
		slim[i] = map[string]any{
			"id":              req.ID,
			"timestamp":       req.Timestamp,
			"service":         req.Service,
			"operation":       req.Operation,
			"method":          req.Method,
			"path":            req.Path,
			"region":          req.Region,
			"matchedScenario": req.MatchedScenario,
			"responseCode":    req.ResponseCode,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"requests": slim,
	})
}

func (cp *ControlPlane) getRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for _, req := range cp.engine.Requests() {
		if req.ID == id {
			writeJSON(w, http.StatusOK, req)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found: " + id})
}

func (cp *ControlPlane) clearRequests(w http.ResponseWriter, r *http.Request) {
	cp.engine.ClearRequests()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// --- Presets ---

func (cp *ControlPlane) applyPreset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	fn, ok := cp.presets[name]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown preset: " + name})
		return
	}

	var config json.RawMessage
	if r.Body != nil && r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&config)
	}

	scenarios, err := fn(config)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	append_ := r.URL.Query().Get("append") == "true"
	if !append_ {
		cp.engine.ClearScenarios()
	}

	var ids []string
	for _, s := range scenarios {
		id := cp.engine.AddScenario(s)
		ids = append(ids, id)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"preset":    name,
		"scenarios": ids,
	})
}

func (cp *ControlPlane) listPresets(w http.ResponseWriter, r *http.Request) {
	names := make([]string, 0, len(cp.presets))
	for name := range cp.presets {
		names = append(names, name)
	}
	writeJSON(w, http.StatusOK, names)
}

// --- Reset ---

func (cp *ControlPlane) reset(w http.ResponseWriter, r *http.Request) {
	cp.engine.Reset()
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func filterByOperation(reqs []engine.RecordedRequest, op string) []engine.RecordedRequest {
	var out []engine.RecordedRequest
	for _, r := range reqs {
		if r.Operation == op {
			out = append(out, r)
		}
	}
	return out
}

func filterByService(reqs []engine.RecordedRequest, svc string) []engine.RecordedRequest {
	var out []engine.RecordedRequest
	for _, r := range reqs {
		if r.Service == svc {
			out = append(out, r)
		}
	}
	return out
}

func filterSince(reqs []engine.RecordedRequest, since time.Time) []engine.RecordedRequest {
	var out []engine.RecordedRequest
	for _, r := range reqs {
		if r.Timestamp.After(since) {
			out = append(out, r)
		}
	}
	return out
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// ApplyPreset applies a named preset to the engine. For Go API parity.
func (cp *ControlPlane) ApplyPreset(name string, config json.RawMessage) error {
	fn, ok := cp.presets[name]
	if !ok {
		return errorf("unknown preset: %s", name)
	}
	scenarios, err := fn(config)
	if err != nil {
		return err
	}
	cp.engine.ClearScenarios()
	for _, s := range scenarios {
		cp.engine.AddScenario(s)
	}
	return nil
}

func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
