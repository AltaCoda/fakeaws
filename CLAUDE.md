# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Mock AWS HTTP server implementing SES v2 and STS APIs. Used by SendOps (sesmail) for testing and local development. Dual-mode: embedded (httptest.Server for Go tests) and standalone (long-running binary with control plane dashboard).

## Commands

```bash
go build ./...                          # build everything
go test ./... -timeout 60s              # run all tests
go test ./internal/engine/ -run TestEngine_STS  # single test
go run ./cmd/fakeaws                    # start standalone on :4579
go run ./cmd/fakeaws --port 5555 --config fakeaws.toml  # custom port + config
```

Docker:
```bash
docker build -t fakeaws .
docker run -p 4579:4579 fakeaws
```

## Architecture

**Public API surface** (root package `fakeaws`):
- `fakeaws.go` — type aliases and re-exports from internal/engine
- `server.go` — `FakeServer` (httptest wrapper), SDK client factories (`SESClient`, `STSClient`), assertion helpers (`AssertCalled`, `AssertCalledN`, `AssertNotCalled`, `AssertFieldEquals`)
- `builder.go` — fluent `ScenarioBuilder` API (`WhenOperation().WithField().RespondThrottle()`)

**Internal packages:**
- `internal/engine/` — core HTTP handler: `Engine` (implements `http.Handler`), `Router` (path-based SES v2 + form-based STS routing), `ScenarioStack` (first-match-wins evaluation), `Recorder` (bounded request log), default handlers, matchers, response writers
- `internal/controlplane/` — HTTP API under `/_control/`: scenario CRUD, request log queries, presets system, JSON-to-Scenario mapping, embedded dashboard HTML (`go:embed`)
- `cmd/fakeaws/` — standalone binary with cobra/viper, `--port`/`--config` flags, graceful shutdown

**Request flow:** HTTP request → `Engine.ServeHTTP` → Router resolves service+operation → ScenarioStack evaluates (first match wins) → if no match, DefaultHandler → Recorder captures request+response

**Unimplemented services** (S3, EventBridge, etc.) get a permissive 200 stub via `permissive.go`. EventBridge `DescribeRule`/`ListTargetsByRule` return realistic stubs detected via `X-Amz-Target` header.

## Key Design Decisions

- All SES v2 response bodies use real AWS SDK output types (`sesv2.SendEmailOutput`, `sestypes.IdentityInfo`, etc.) to guarantee wire compatibility. Never use `map[string]any` for AWS responses.
- STS uses XML wire format (stdlib `encoding/xml`). SES v2 uses JSON.
- `Engine` fields (`scenarios`, `recorder`, `router`) are unexported. Access through public methods only.
- `Scenario.Delay` is context-aware (`select` with `time.After` + `req.Context.Done()`).
- `ParsedRequest.Context` carries the original `*http.Request` context for timeout-aware responders.
- The `Probability(p)` matcher uses `math/rand` (auto-seeded in Go 1.20+).

## Adding New AWS Operations

1. Add route to `internal/engine/routes_ses.go` (or `routes_sts.go`)
2. Add default handler in `internal/engine/defaults.go` using the real SDK output type from `github.com/aws/aws-sdk-go-v2/service/sesv2`
3. If the operation needs specific happy_path behavior, update `internal/controlplane/presets.go`

## Integration with SendOps

This repo is used by `github.com/AltaCoda/sesmail`:
- Docker image at `ghcr.io/altacoda/fakeaws:latest` (multi-arch: amd64+arm64)
- SendOps sets `AWS_ENDPOINT_URL=http://localhost:4579` (host mode) or `http://fakeaws:4579` (Docker mode)
- The Go SDK v2's `config.LoadDefaultConfig` natively respects `AWS_ENDPOINT_URL`
- SendOps CI uses fakeaws as a GitHub Actions service container
- Fakeaws tests in sesmail are at `backend/internal/preflight/preflight_fakeaws_test.go` and `backend/internal/mailer/ses_fakeaws_test.go`
