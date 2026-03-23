// Package fakeaws provides a mock AWS server for SES v2 and STS.
//
// Use NewEngine to create the core HTTP handler, then mount it on an
// httptest.Server (embedded mode) or a real listener (standalone mode).
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
)
