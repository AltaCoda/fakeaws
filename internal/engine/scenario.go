package engine

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Matcher tests whether a ParsedRequest should trigger a scenario.
type Matcher func(req *ParsedRequest) bool

// Responder writes an HTTP response for a matched request.
type Responder func(w http.ResponseWriter, req *ParsedRequest)

// Scenario defines a rule: if Matcher returns true, Responder writes the response.
type Scenario struct {
	ID                  string
	Name                string
	Matcher             Matcher
	Responder           Responder
	Once                bool // auto-remove after first match
	Delay               time.Duration
	HitCount            atomic.Int64
	CreatedAt           time.Time
	MatcherDescription  string
	ResponseDescription string
}

// ScenarioInfo is a serializable view of a Scenario for the control plane.
type ScenarioInfo struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Once                bool   `json:"once"`
	HitCount            int64  `json:"hitCount"`
	CreatedAt           time.Time `json:"createdAt"`
	MatcherDescription  string `json:"matcherDescription"`
	ResponseDescription string `json:"responseDescription"`
}

func (s *Scenario) Info() ScenarioInfo {
	return ScenarioInfo{
		ID:                  s.ID,
		Name:                s.Name,
		Once:                s.Once,
		HitCount:            s.HitCount.Load(),
		CreatedAt:           s.CreatedAt,
		MatcherDescription:  s.MatcherDescription,
		ResponseDescription: s.ResponseDescription,
	}
}

// ScenarioStack is a thread-safe ordered list of scenarios.
// Evaluation walks top-down; first match wins.
type ScenarioStack struct {
	mu        sync.RWMutex
	scenarios []*Scenario
	counter   int64
}

// Push adds a scenario to the top of the stack and returns its auto-generated ID.
func (ss *ScenarioStack) Push(s *Scenario) string {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.counter++
	s.ID = fmt.Sprintf("sc_%03d", ss.counter)
	s.CreatedAt = time.Now()

	// Prepend — newest scenario is first (highest priority)
	ss.scenarios = append([]*Scenario{s}, ss.scenarios...)
	return s.ID
}

// Remove deletes a scenario by ID. Returns true if found.
func (ss *ScenarioStack) Remove(id string) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	for i, s := range ss.scenarios {
		if s.ID == id {
			ss.scenarios = append(ss.scenarios[:i], ss.scenarios[i+1:]...)
			return true
		}
	}
	return false
}

// List returns info for all active scenarios.
func (ss *ScenarioStack) List() []ScenarioInfo {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	infos := make([]ScenarioInfo, len(ss.scenarios))
	for i, s := range ss.scenarios {
		infos[i] = s.Info()
	}
	return infos
}

// Clear removes all scenarios.
func (ss *ScenarioStack) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.scenarios = nil
}

// Evaluate walks the stack top-down. First match wins.
// Returns the responder and scenario ID, or nil/"" if no match.
func (ss *ScenarioStack) Evaluate(req *ParsedRequest) (Responder, string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	for i, s := range ss.scenarios {
		if s.Matcher(req) {
			s.HitCount.Add(1)

			responder := s.Responder
			id := s.ID

			if s.Once {
				ss.scenarios = append(ss.scenarios[:i], ss.scenarios[i+1:]...)
			}

			if s.Delay > 0 {
				original := responder
				delay := s.Delay
				responder = func(w http.ResponseWriter, req *ParsedRequest) {
					select {
					case <-time.After(delay):
						original(w, req)
					case <-req.Context.Done():
					}
				}
			}

			return responder, id
		}
	}

	return nil, ""
}
