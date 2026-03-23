// Package fakeaws provides a mock AWS server for SES v2 and STS.
//
// For tests, use NewFakeServer() which wraps the engine in an httptest.Server
// and provides SDK client factories:
//
//	fake := fakeaws.NewFakeServer()
//	defer fake.Close()
//	client := fake.SESClient(ctx)
//
// Use the builder API for ergonomic scenario setup:
//
//	fake.AddScenario(fakeaws.WhenOperation("SendEmail").RespondThrottle())
//
// For standalone mode, use NewEngine directly with a real listener.
package fakeaws

import (
	"github.com/altacoda/fakeaws/internal/engine"
	"go.uber.org/zap"
)

// Re-export core types for library consumers.
type (
	Engine          = engine.Engine
	Scenario        = engine.Scenario
	ScenarioInfo    = engine.ScenarioInfo
	Matcher         = engine.Matcher
	Responder       = engine.Responder
	ParsedRequest   = engine.ParsedRequest
	RecordedRequest = engine.RecordedRequest
)

// NewEngine creates a new fake AWS engine.
func NewEngine(logger *zap.Logger) *Engine {
	return engine.NewEngine(logger)
}

// --- Matchers ---

var (
	OperationIs  = engine.OperationIs
	ServiceIs    = engine.ServiceIs
	FieldEquals  = engine.FieldEquals
	HeaderEquals = engine.HeaderEquals
	All          = engine.All
	Any          = engine.Any
	Probability  = engine.Probability
)

// --- Constants ---

const (
	FakeAccountID   = engine.FakeAccountID
	FakeAccessKeyID = engine.FakeAccessKeyID
	FakeSecretKey   = engine.FakeSecretKey
	FakeRegion       = engine.FakeRegion
	FakeSessionToken = engine.FakeSessionToken
)

// --- Utilities ---

var (
	GenerateMessageID = engine.GenerateMessageID
	GenerateARN       = engine.GenerateARN
	FakeCredentials   = engine.FakeCredentials
)

// --- Response helpers for custom responders ---

var (
	WriteJSONResponse = engine.WriteJSONResponse
	WriteJSONError    = engine.WriteJSONError
	WriteXMLResponse  = engine.WriteXMLResponse
	WriteXMLError     = engine.WriteXMLError
	WriteError        = engine.WriteError
)
