package engine

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const DefaultMaxRequests = 1000

// RecordedRequest captures everything about a single request/response cycle.
type RecordedRequest struct {
	ID              string         `json:"id"`
	Timestamp       time.Time      `json:"timestamp"`
	Service         string         `json:"service"`
	Operation       string         `json:"operation"`
	Method          string         `json:"method"`
	Path            string         `json:"path"`
	Headers         http.Header    `json:"headers,omitempty"`
	Body            map[string]any `json:"body,omitempty"`
	Region          string         `json:"region"`
	AccessKeyID     string         `json:"accessKeyId"`
	MatchedScenario string         `json:"matchedScenario,omitempty"`
	ResponseCode    int            `json:"responseCode"`
	ResponseBody    string         `json:"responseBody,omitempty"`
}

// Recorder stores request history with a bounded buffer.
type Recorder struct {
	mu          sync.Mutex
	requests    []RecordedRequest
	maxRequests int
	counter     int64
}

// NewRecorder creates a Recorder with the given capacity.
func NewRecorder(maxRequests int) *Recorder {
	if maxRequests <= 0 {
		maxRequests = DefaultMaxRequests
	}
	return &Recorder{
		maxRequests: maxRequests,
	}
}

// Append adds a request to the log, evicting the oldest if over capacity.
func (r *Recorder) Append(req RecordedRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counter++
	req.ID = fmt.Sprintf("req_%03d", r.counter)

	r.requests = append(r.requests, req)
	if len(r.requests) > r.maxRequests {
		r.requests = r.requests[len(r.requests)-r.maxRequests:]
	}
}

// All returns all recorded requests.
func (r *Recorder) All() []RecordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RecordedRequest, len(r.requests))
	copy(out, r.requests)
	return out
}

// For returns requests matching the given service and operation.
func (r *Recorder) For(service, operation string) []RecordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []RecordedRequest
	for _, req := range r.requests {
		if req.Service == service && req.Operation == operation {
			out = append(out, req)
		}
	}
	return out
}

// Count returns the number of requests matching service and operation.
func (r *Recorder) Count(service, operation string) int {
	return len(r.For(service, operation))
}

// Last returns the most recent request matching service and operation, or nil.
func (r *Recorder) Last(service, operation string) *RecordedRequest {
	reqs := r.For(service, operation)
	if len(reqs) == 0 {
		return nil
	}
	return &reqs[len(reqs)-1]
}

// Clear removes all recorded requests.
func (r *Recorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = nil
}
