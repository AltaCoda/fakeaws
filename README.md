# fakeaws

Mock AWS server implementing SES v2 and STS APIs. Built for testing and local development.

Two modes: **embedded** (in-process via `httptest.Server` for Go tests) and **standalone** (long-running with HTTP control plane and dashboard).

## Quick Start: Embedded Mode

```go
func TestSendEmail(t *testing.T) {
    fake := fakeaws.NewFakeServer()
    defer fake.Close()

    ctx := context.Background()
    client := fake.SESClient(ctx)

    out, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
        FromEmailAddress: aws.String("sender@example.com"),
        Destination:      &types.Destination{ToAddresses: []string{"to@example.com"}},
        Content: &types.EmailContent{
            Simple: &types.Message{
                Subject: &types.Content{Data: aws.String("Test")},
                Body:    &types.Body{Text: &types.Content{Data: aws.String("Hello")}},
            },
        },
    })
    require.NoError(t, err)
    assert.NotEmpty(t, out.MessageId)

    fake.AssertCalled(t, "sesv2", "SendEmail")
    fake.AssertCalledN(t, "sesv2", "SendEmail", 1)
}
```

## Quick Start: Standalone Mode

```bash
go run ./cmd/fakeaws
# fakeaws listening on :4579
#   AWS endpoint:   http://localhost:4579
#   Control plane:  http://localhost:4579/_control/dashboard
```

Point your AWS SDK at it:

```bash
export AWS_ENDPOINT_URL=http://localhost:4579
```

## Builder API

Fluent API for scenario setup in tests:

```go
// Throttle all sends
fake.AddScenario(fakeaws.WhenOperation("SendEmail").RespondThrottle())

// Reject specific addresses
fake.AddScenario(
    fakeaws.WhenOperation("SendEmail").
        WithField("Destination.ToAddresses[0]", "blocked@example.com").
        RespondMessageRejected("Address is blocked"),
)

// One-time error then success
fake.AddScenario(
    fakeaws.WhenOperation("SendEmail").Once().RespondThrottle(),
)

// Simulate slow responses
fake.AddScenario(
    fakeaws.WhenOperation("GetAccount").
        RespondAfter(2 * time.Second).
        RespondSuccess(map[string]any{"SendingEnabled": true}),
)

// Timeout (blocks until client cancels)
fake.AddScenario(fakeaws.WhenOperation("SendEmail").RespondTimeout())
```

## Assertion Helpers

```go
fake.AssertCalled(t, "sesv2", "SendEmail")
fake.AssertCalledN(t, "sesv2", "SendEmail", 3)
fake.AssertNotCalled(t, "sts", "AssumeRole")

reqs := fake.RequestsFor("sesv2", "SendEmail")
fake.AssertFieldEquals(t, reqs[0], "FromEmailAddress", "sender@example.com")
```

## Presets

Apply pre-configured scenario bundles:

```go
// Go API
fake.ApplyPreset("throttle_all_sends", nil)
fake.ApplyPreset("sandbox_mode", map[string]any{
    "verified_addresses": []string{"test@sendops.dev"},
})
```

```bash
# HTTP API
curl -X POST http://localhost:4579/_control/scenarios/presets/throttle_all_sends
curl -X POST http://localhost:4579/_control/scenarios/presets/sandbox_mode \
  -d '{"verified_addresses": ["test@sendops.dev"]}'
```

Available presets:

| Preset | Effect |
|--------|--------|
| `happy_path` | All defaults (everything succeeds) |
| `sending_paused` | GetAccount returns SendingEnabled=false |
| `throttle_all_sends` | SendEmail returns 429 |
| `reject_unverified` | SendEmail rejects addresses not in allowlist (requires `verified_addresses`) |
| `sandbox_mode` | SendEmail only succeeds for verified addresses (requires `verified_addresses`) |
| `intermittent_failures` | 20% of SendEmail calls return 500 |
| `slow_responses` | All responses delayed 2 seconds |

## Control Plane API

All endpoints under `/_control/`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/_control/scenarios` | List active scenarios |
| `POST` | `/_control/scenarios` | Create scenario from JSON |
| `DELETE` | `/_control/scenarios/{id}` | Remove scenario |
| `GET` | `/_control/requests` | List recorded requests |
| `GET` | `/_control/requests/{id}` | Request detail |
| `DELETE` | `/_control/requests` | Clear request log |
| `POST` | `/_control/scenarios/presets/{name}` | Apply preset |
| `GET` | `/_control/presets` | List available presets |
| `POST` | `/_control/reset` | Clear everything |
| `GET` | `/_control/dashboard` | Dashboard UI |

Request log filters: `?operation=SendEmail`, `?service=sesv2`, `?last=10`, `?since=2025-01-01T00:00:00Z`

## Supported Operations

**SES v2** (JSON over REST):
SendEmail, SendBulkEmail, CreateConfigurationSet, GetConfigurationSet, DeleteConfigurationSet, GetAccount, PutAccountSendingAttributes, CreateEmailIdentity, GetEmailIdentity, DeleteEmailIdentity

**STS** (XML over POST):
AssumeRole, GetCallerIdentity

## Matchers

```go
fakeaws.OperationIs("SendEmail")
fakeaws.ServiceIs("sesv2")
fakeaws.FieldEquals("Destination.ToAddresses[0]", "user@example.com")
fakeaws.HeaderEquals("X-Custom", "value")
fakeaws.All(matcher1, matcher2)       // AND
fakeaws.Any(matcher1, matcher2)       // OR
fakeaws.Probability(0.2)             // 20% match rate
```

## Configuration File

Standalone mode supports TOML config via `--config`:

```toml
preset = "sandbox_mode"

[preset_config]
verified_addresses = ["test@sendops.dev"]

[[scenarios]]
name = "Slow sends"
operation = "SendEmail"
[scenarios.response]
type = "delay"
delay_ms = 500

[[scenarios]]
name = "Reject blocked domain"
operation = "SendEmail"
[scenarios.match_fields]
"Destination.ToAddresses[0]" = "user@blocked.com"
[scenarios.response]
type = "error"
status_code = 400
error_code = "MessageRejected"
message = "Address is suppressed"
```

## Architecture

```
fakeaws.go              → Public API (type aliases, re-exports)
server.go               → FakeServer, SDK client factories, assertions
builder.go              → Fluent scenario builder API
internal/engine/        → Core: Engine, Router, Scenarios, Recorder, Defaults
internal/controlplane/  → HTTP control plane, presets, dashboard
cmd/fakeaws/            → Standalone binary (cobra + viper)
```
