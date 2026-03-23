package controlplane

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/altacoda/fakeaws/internal/engine"
)

// ScenarioInput is the JSON-serializable input for creating a scenario via the control plane.
type ScenarioInput struct {
	Name         string            `json:"name"`
	Operation    string            `json:"operation,omitempty"`
	Service      string            `json:"service,omitempty"`
	MatchFields  map[string]string `json:"match_fields,omitempty"`
	MatchHeaders map[string]string `json:"match_headers,omitempty"`
	Response     ResponseSpec      `json:"response"`
	Once         bool              `json:"once,omitempty"`
}

// ResponseSpec describes the response a scenario should produce.
type ResponseSpec struct {
	Type        string `json:"type"` // success, error, throttle, timeout, delay
	StatusCode  int    `json:"status_code,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	Message     string `json:"message,omitempty"`
	Body        any    `json:"body,omitempty"`
	DelayMs     int    `json:"delay_ms,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// ScenarioFromJSON converts a ScenarioInput into a Scenario.
func ScenarioFromJSON(input ScenarioInput) (*engine.Scenario, error) {
	// Build matchers
	var matchers []engine.Matcher
	var descs []string

	if input.Operation != "" {
		matchers = append(matchers, engine.OperationIs(input.Operation))
		descs = append(descs, "operation="+input.Operation)
	}
	if input.Service != "" {
		matchers = append(matchers, engine.ServiceIs(input.Service))
		descs = append(descs, "service="+input.Service)
	}
	for path, value := range input.MatchFields {
		matchers = append(matchers, engine.FieldEquals(path, value))
		descs = append(descs, fmt.Sprintf("%s=%q", path, value))
	}
	for key, value := range input.MatchHeaders {
		matchers = append(matchers, engine.HeaderEquals(key, value))
		descs = append(descs, fmt.Sprintf("header[%s]=%q", key, value))
	}

	// Default matcher: match everything
	var matcher engine.Matcher
	if len(matchers) == 0 {
		matcher = func(req *engine.ParsedRequest) bool { return true }
		descs = append(descs, "*")
	} else {
		matcher = engine.All(matchers...)
	}

	// Build responder
	responder, responseDesc, err := buildResponder(input.Response)
	if err != nil {
		return nil, err
	}

	name := input.Name
	if name == "" {
		name = strings.Join(descs, " AND ") + " → " + responseDesc
	}

	s := &engine.Scenario{
		Name:                name,
		Matcher:             matcher,
		Responder:           responder,
		Once:                input.Once,
		MatcherDescription:  strings.Join(descs, " AND "),
		ResponseDescription: responseDesc,
	}

	if input.Response.DelayMs > 0 {
		s.Delay = time.Duration(input.Response.DelayMs) * time.Millisecond
	}

	return s, nil
}

func buildResponder(spec ResponseSpec) (engine.Responder, string, error) {
	switch spec.Type {
	case "success", "":
		if spec.Body != nil {
			body := spec.Body
			return func(w http.ResponseWriter, req *engine.ParsedRequest) {
				code := spec.StatusCode
				if code == 0 {
					code = http.StatusOK
				}
				engine.WriteJSONResponse(w, code, body)
			}, "success (custom body)", nil
		}
		// No body specified — delegate to default handler for SDK-typed response
		return func(w http.ResponseWriter, req *engine.ParsedRequest) {
			engine.DefaultHandler(req.Operation)(w, req)
		}, "success (default)", nil

	case "error":
		if spec.ErrorCode == "" {
			return nil, "", fmt.Errorf("error response requires error_code")
		}
		code := spec.StatusCode
		if code == 0 {
			code = http.StatusBadRequest
		}
		msg := spec.Message
		if msg == "" {
			msg = spec.ErrorCode
		}
		return func(w http.ResponseWriter, req *engine.ParsedRequest) {
			engine.WriteError(w, req.Service, code, spec.ErrorCode, msg)
		}, fmt.Sprintf("error %d %s", code, spec.ErrorCode), nil

	case "throttle":
		return func(w http.ResponseWriter, req *engine.ParsedRequest) {
			engine.WriteError(w, req.Service, http.StatusTooManyRequests, "TooManyRequestsException", "Rate exceeded")
		}, "throttle 429", nil

	case "timeout":
		return func(w http.ResponseWriter, req *engine.ParsedRequest) {
			<-req.Context.Done()
		}, "timeout", nil

	case "delay":
		// Delay is handled via Scenario.Delay field, responder delegates to default
		if spec.Body != nil {
			body := spec.Body
			return func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteJSONResponse(w, http.StatusOK, body)
			}, fmt.Sprintf("delay %dms (custom body)", spec.DelayMs), nil
		}
		return func(w http.ResponseWriter, req *engine.ParsedRequest) {
			engine.DefaultHandler(req.Operation)(w, req)
		}, fmt.Sprintf("delay %dms", spec.DelayMs), nil

	default:
		return nil, "", fmt.Errorf("unknown response type: %q", spec.Type)
	}
}
