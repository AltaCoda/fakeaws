package fakeaws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.uber.org/zap"

	"github.com/altacoda/fakeaws/internal/controlplane"
	"github.com/altacoda/fakeaws/internal/engine"
)

// FakeServer wraps an Engine in an httptest.Server for in-process test usage.
type FakeServer struct {
	engine       *engine.Engine
	controlPlane *controlplane.ControlPlane
	server       *httptest.Server
}

// NewFakeServer creates a FakeServer with a running httptest.Server.
func NewFakeServer() *FakeServer {
	return NewFakeServerWithLogger(zap.NewNop())
}

// NewFakeServerWithLogger creates a FakeServer with a custom logger.
func NewFakeServerWithLogger(logger *zap.Logger) *FakeServer {
	e := engine.NewEngine(logger)
	cp := controlplane.New(e, logger)

	mux := http.NewServeMux()
	mux.Handle("/_control/", cp.Handler())
	mux.Handle("/", e)

	srv := httptest.NewServer(mux)

	return &FakeServer{
		engine:       e,
		controlPlane: cp,
		server:       srv,
	}
}

// URL returns the base URL of the fake server.
func (fs *FakeServer) URL() string {
	return fs.server.URL
}

// Close stops the server. Safe for defer.
func (fs *FakeServer) Close() {
	fs.server.Close()
}

// Engine returns direct access to the underlying engine.
func (fs *FakeServer) Engine() *Engine {
	return fs.engine
}

// --- Convenience delegations ---

// AddScenario pushes a scenario onto the stack.
func (fs *FakeServer) AddScenario(s *Scenario) string {
	return fs.engine.AddScenario(s)
}

// RemoveScenario removes a scenario by ID.
func (fs *FakeServer) RemoveScenario(id string) bool {
	return fs.engine.RemoveScenario(id)
}

// Reset clears all scenarios and recorded requests.
func (fs *FakeServer) Reset() {
	fs.engine.Reset()
}

// Requests returns all recorded requests.
func (fs *FakeServer) Requests() []RecordedRequest {
	return fs.engine.Requests()
}

// RequestsFor returns requests matching service and operation.
func (fs *FakeServer) RequestsFor(service, operation string) []RecordedRequest {
	return fs.engine.RequestsFor(service, operation)
}

// CountRequests returns the count of requests matching service and operation.
func (fs *FakeServer) CountRequests(service, operation string) int {
	return fs.engine.CountRequests(service, operation)
}

// LastRequest returns the most recent request matching service and operation.
func (fs *FakeServer) LastRequest(service, operation string) *RecordedRequest {
	return fs.engine.LastRequest(service, operation)
}

// ApplyPreset applies a named preset to the engine.
func (fs *FakeServer) ApplyPreset(name string, config any) error {
	var configJSON []byte
	if config != nil {
		var err error
		configJSON, err = json.Marshal(config)
		if err != nil {
			return err
		}
	}
	return fs.controlPlane.ApplyPreset(name, configJSON)
}

// --- SDK Client Factories ---

// mustAWSConfig returns an aws.Config with fake credentials. Panics on failure
// (which should not happen with static credentials).
func (fs *FakeServer) mustAWSConfig(ctx context.Context) aws.Config {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(FakeRegion),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(FakeAccessKeyID, FakeSecretKey, ""),
		),
		awsconfig.WithRetryMaxAttempts(1),
	)
	if err != nil {
		panic("fakeaws: failed to load AWS config: " + err.Error())
	}
	return cfg
}

// SESClient creates an SES v2 client pointed at the fake server.
func (fs *FakeServer) SESClient(ctx context.Context) *sesv2.Client {
	cfg := fs.mustAWSConfig(ctx)
	return sesv2.NewFromConfig(cfg, func(o *sesv2.Options) {
		o.BaseEndpoint = aws.String(fs.server.URL)
	})
}

// STSClient creates an STS client pointed at the fake server.
func (fs *FakeServer) STSClient(ctx context.Context) *sts.Client {
	cfg := fs.mustAWSConfig(ctx)
	return sts.NewFromConfig(cfg, func(o *sts.Options) {
		o.BaseEndpoint = aws.String(fs.server.URL)
	})
}

// --- Assertion Helpers ---

// AssertCalled fails the test if the given operation was never called.
func (fs *FakeServer) AssertCalled(t testing.TB, service, operation string) {
	t.Helper()
	count := fs.engine.CountRequests(service, operation)
	if count == 0 {
		t.Errorf("expected %s.%s to have been called, but it was not\nRecorded operations: %s",
			service, operation, fs.recordedOperationsSummary())
	}
}

// AssertCalledN fails the test if the call count doesn't match.
func (fs *FakeServer) AssertCalledN(t testing.TB, service, operation string, n int) {
	t.Helper()
	count := fs.engine.CountRequests(service, operation)
	if count != n {
		t.Errorf("expected %s.%s to have been called %d time(s), but was called %d time(s)\nRecorded operations: %s",
			service, operation, n, count, fs.recordedOperationsSummary())
	}
}

// AssertNotCalled fails the test if the given operation was called.
func (fs *FakeServer) AssertNotCalled(t testing.TB, service, operation string) {
	t.Helper()
	count := fs.engine.CountRequests(service, operation)
	if count > 0 {
		t.Errorf("expected %s.%s to not have been called, but it was called %d time(s)",
			service, operation, count)
	}
}

// AssertFieldEquals fails if the recorded request's field at the given dot-path
// doesn't match the expected value. Only checks the parsed Body map.
func (fs *FakeServer) AssertFieldEquals(t testing.TB, req RecordedRequest, path string, expected any) {
	t.Helper()
	pr := &ParsedRequest{Body: req.Body}
	val, ok := pr.FieldAt(path)
	if !ok {
		t.Errorf("field %q not found in recorded request %s.%s (id=%s)",
			path, req.Service, req.Operation, req.ID)
		return
	}
	fmtVal := fmt.Sprintf("%v", val)
	expVal := fmt.Sprintf("%v", expected)
	if fmtVal != expVal {
		t.Errorf("field %q: expected %s, got %s (request %s.%s id=%s)",
			path, expVal, fmtVal, req.Service, req.Operation, req.ID)
	}
}

func (fs *FakeServer) recordedOperationsSummary() string {
	reqs := fs.engine.Requests()
	if len(reqs) == 0 {
		return "(none)"
	}
	seen := make(map[string]int)
	for _, r := range reqs {
		key := r.Service + "." + r.Operation
		seen[key]++
	}
	var parts []string
	for k, v := range seen {
		parts = append(parts, k+"("+strconv.Itoa(v)+")")
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
