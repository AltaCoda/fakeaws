package fakeaws

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/altacoda/fakeaws/internal/engine"
)

// ScenarioBuilder provides a fluent API for constructing scenarios.
type ScenarioBuilder struct {
	name     string
	matchers []Matcher
	descs    []string // matcher descriptions
	once     bool
	delay    time.Duration
}

// WhenOperation starts building a scenario that matches a specific operation.
func WhenOperation(op string) *ScenarioBuilder {
	return &ScenarioBuilder{
		matchers: []Matcher{engine.OperationIs(op)},
		descs:    []string{"operation=" + op},
	}
}

// When starts building a scenario with explicit matchers.
func When(matchers ...Matcher) *ScenarioBuilder {
	return &ScenarioBuilder{
		matchers: matchers,
		descs:    []string{"custom matchers"},
	}
}

// Named sets a human-readable name for the scenario.
func (b *ScenarioBuilder) Named(name string) *ScenarioBuilder {
	b.name = name
	return b
}

// WithField adds a field equality matcher.
func (b *ScenarioBuilder) WithField(path, value string) *ScenarioBuilder {
	b.matchers = append(b.matchers, engine.FieldEquals(path, value))
	b.descs = append(b.descs, fmt.Sprintf("%s=%q", path, value))
	return b
}

// WithHeader adds a header equality matcher.
func (b *ScenarioBuilder) WithHeader(key, value string) *ScenarioBuilder {
	b.matchers = append(b.matchers, engine.HeaderEquals(key, value))
	b.descs = append(b.descs, fmt.Sprintf("header[%s]=%q", key, value))
	return b
}

// Where adds a custom predicate matcher.
func (b *ScenarioBuilder) Where(fn func(*ParsedRequest) bool) *ScenarioBuilder {
	b.matchers = append(b.matchers, fn)
	b.descs = append(b.descs, "custom predicate")
	return b
}

// Once marks the scenario to auto-remove after first match.
func (b *ScenarioBuilder) Once() *ScenarioBuilder {
	b.once = true
	return b
}

// RespondAfter adds a delay before the response. Chain with a Respond* method.
func (b *ScenarioBuilder) RespondAfter(d time.Duration) *ScenarioBuilder {
	b.delay = d
	return b
}

// RespondSuccess builds a scenario returning a JSON success response.
func (b *ScenarioBuilder) RespondSuccess(body any) *Scenario {
	return b.build("success", func(w http.ResponseWriter, req *ParsedRequest) {
		engine.WriteJSONResponse(w, http.StatusOK, body)
	})
}

// RespondError builds a scenario returning an error response.
func (b *ScenarioBuilder) RespondError(code int, errorCode, message string) *Scenario {
	return b.build(fmt.Sprintf("error %d %s", code, errorCode), func(w http.ResponseWriter, req *ParsedRequest) {
		engine.WriteError(w, req.Service, code, errorCode, message)
	})
}

// RespondWith builds a scenario with a custom responder.
func (b *ScenarioBuilder) RespondWith(fn Responder) *Scenario {
	return b.build("custom responder", fn)
}

// RespondTimeout builds a scenario that blocks until the client disconnects or context cancels.
func (b *ScenarioBuilder) RespondTimeout() *Scenario {
	return b.build("timeout", func(w http.ResponseWriter, req *ParsedRequest) {
		<-req.Context.Done()
	})
}

// --- Convenience error responders ---

// RespondMessageRejected returns a 400 MessageRejected error.
func (b *ScenarioBuilder) RespondMessageRejected(msg string) *Scenario {
	return b.RespondError(http.StatusBadRequest, "MessageRejected", msg)
}

// RespondThrottle returns a 429 TooManyRequestsException error.
func (b *ScenarioBuilder) RespondThrottle() *Scenario {
	return b.RespondError(http.StatusTooManyRequests, "TooManyRequestsException", "Rate exceeded")
}

// RespondAccountSendingPaused returns a 400 AccountSendingPausedException error.
func (b *ScenarioBuilder) RespondAccountSendingPaused() *Scenario {
	return b.RespondError(http.StatusBadRequest, "AccountSendingPausedException", "Account sending is currently paused")
}

// RespondMailFromNotVerified returns a 400 MailFromDomainNotVerifiedException error.
func (b *ScenarioBuilder) RespondMailFromNotVerified() *Scenario {
	return b.RespondError(http.StatusBadRequest, "MailFromDomainNotVerifiedException", "Mail from domain not verified")
}

// RespondConfigurationSetNotFound returns a 404 NotFoundException error.
func (b *ScenarioBuilder) RespondConfigurationSetNotFound() *Scenario {
	return b.RespondError(http.StatusNotFound, "NotFoundException", "Configuration set not found")
}

// --- Internal ---

func (b *ScenarioBuilder) build(responseDesc string, responder Responder) *Scenario {
	matcher := engine.All(b.matchers...)
	matcherDesc := strings.Join(b.descs, " AND ")

	name := b.name
	if name == "" {
		name = matcherDesc
	}

	return &Scenario{
		Name:                name,
		Matcher:             matcher,
		Responder:           responder,
		Once:                b.once,
		Delay:               b.delay,
		MatcherDescription:  matcherDesc,
		ResponseDescription: responseDesc,
	}
}
