package controlplane

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/altacoda/fakeaws/internal/engine"
)

// PresetFunc creates scenarios from optional config JSON.
type PresetFunc func(config json.RawMessage) ([]*engine.Scenario, error)

func builtinPresets() map[string]PresetFunc {
	return map[string]PresetFunc{
		"happy_path":            presetHappyPath,
		"sending_paused":        presetSendingPaused,
		"throttle_all_sends":    presetThrottleAllSends,
		"reject_unverified":     presetRejectUnverified,
		"sandbox_mode":          presetSandboxMode,
		"intermittent_failures": presetIntermittentFailures,
		"slow_responses":        presetSlowResponses,
	}
}

func presetHappyPath(_ json.RawMessage) ([]*engine.Scenario, error) {
	return []*engine.Scenario{
		{
			Name:                "happy_path: ListEmailIdentities with fake identities",
			Matcher:             engine.OperationIs("ListEmailIdentities"),
			MatcherDescription:  "operation=ListEmailIdentities",
			ResponseDescription: "fake verified identities",
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteJSONResponse(w, http.StatusOK, map[string]any{
					"EmailIdentities": []map[string]any{
						{
							"IdentityType":             "DOMAIN",
							"IdentityName":             "sendops.dev",
							"SendingEnabled":           true,
							"VerificationStatus":       "SUCCESS",
						},
						{
							"IdentityType":             "DOMAIN",
							"IdentityName":             "example.com",
							"SendingEnabled":           true,
							"VerificationStatus":       "SUCCESS",
						},
						{
							"IdentityType":             "EMAIL_ADDRESS",
							"IdentityName":             "noreply@sendops.dev",
							"SendingEnabled":           true,
							"VerificationStatus":       "SUCCESS",
						},
						{
							"IdentityType":             "EMAIL_ADDRESS",
							"IdentityName":             "test@example.com",
							"SendingEnabled":           true,
							"VerificationStatus":       "SUCCESS",
						},
					},
				})
			},
		},
		{
			Name:                "happy_path: GetEmailIdentity returns verified with DKIM",
			Matcher:             engine.OperationIs("GetEmailIdentity"),
			MatcherDescription:  "operation=GetEmailIdentity",
			ResponseDescription: "verified identity with DKIM",
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				identity := req.PathParams["EmailIdentity"]
				identityType := "EMAIL_ADDRESS"
				if !strings.Contains(identity, "@") {
					identityType = "DOMAIN"
				}
				engine.WriteJSONResponse(w, http.StatusOK, map[string]any{
					"IdentityType":             identityType,
					"VerifiedForSendingStatus": true,
					"FeedbackForwardingStatus": true,
					"DkimAttributes": map[string]any{
						"SigningEnabled": true,
						"Status":         "SUCCESS",
						"Tokens":         []string{"dkim1-" + identity, "dkim2-" + identity, "dkim3-" + identity},
					},
				})
			},
		},
		{
			Name:                "happy_path: ListConfigurationSets with default set",
			Matcher:             engine.OperationIs("ListConfigurationSets"),
			MatcherDescription:  "operation=ListConfigurationSets",
			ResponseDescription: "default configuration set",
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteJSONResponse(w, http.StatusOK, map[string]any{
					"ConfigurationSets": []string{"sendops-events"},
				})
			},
		},
	}, nil
}


func presetSendingPaused(_ json.RawMessage) ([]*engine.Scenario, error) {
	return []*engine.Scenario{
		{
			Name:                "sending_paused: GetAccount returns SendingEnabled=false",
			Matcher:             engine.OperationIs("GetAccount"),
			MatcherDescription:  "operation=GetAccount",
			ResponseDescription: "SendingEnabled=false",
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteJSONResponse(w, http.StatusOK, map[string]any{
					"SendQuota": map[string]any{
						"Max24HourSend":   50000.0,
						"MaxSendRate":     14.0,
						"SentLast24Hours": 0.0,
					},
					"SendingEnabled": false,
				})
			},
		},
	}, nil
}

func presetThrottleAllSends(_ json.RawMessage) ([]*engine.Scenario, error) {
	return []*engine.Scenario{
		{
			Name:                "throttle_all_sends: SendEmail → 429",
			Matcher:             engine.OperationIs("SendEmail"),
			MatcherDescription:  "operation=SendEmail",
			ResponseDescription: "throttle 429",
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteError(w, req.Service, http.StatusTooManyRequests, "TooManyRequestsException", "Rate exceeded")
			},
		},
	}, nil
}

type verifiedConfig struct {
	VerifiedAddresses []string `json:"verified_addresses"`
}

func presetRejectUnverified(config json.RawMessage) ([]*engine.Scenario, error) {
	var cfg verifiedConfig
	if len(config) > 0 {
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, fmt.Errorf("reject_unverified config: %w", err)
		}
	}
	if len(cfg.VerifiedAddresses) == 0 {
		return nil, fmt.Errorf("reject_unverified requires verified_addresses")
	}

	allowed := make(map[string]bool)
	for _, addr := range cfg.VerifiedAddresses {
		allowed[addr] = true
	}

	return []*engine.Scenario{
		{
			Name:               "reject_unverified: reject sends from unverified addresses",
			MatcherDescription: "operation=SendEmail AND from not in allowlist",
			ResponseDescription: "403 MessageRejected",
			Matcher: func(req *engine.ParsedRequest) bool {
				if req.Operation != "SendEmail" {
					return false
				}
				from := req.FieldString("FromEmailAddress")
				return !allowed[from]
			},
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteError(w, req.Service, http.StatusForbidden, "MessageRejected", "Email address is not verified")
			},
		},
	}, nil
}

func presetSandboxMode(config json.RawMessage) ([]*engine.Scenario, error) {
	var cfg verifiedConfig
	if len(config) > 0 {
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, fmt.Errorf("sandbox_mode config: %w", err)
		}
	}
	if len(cfg.VerifiedAddresses) == 0 {
		return nil, fmt.Errorf("sandbox_mode requires verified_addresses")
	}

	allowed := make(map[string]bool)
	for _, addr := range cfg.VerifiedAddresses {
		allowed[addr] = true
	}

	return []*engine.Scenario{
		{
			// Note: only checks the first To address. Real SES sandbox blocks
			// the entire send if any recipient is unverified. This is a simplification
			// sufficient for testing the common single-recipient path.
			Name:                "sandbox_mode: only allow sends to verified addresses",
			MatcherDescription:  "operation=SendEmail AND to not in verified list",
			ResponseDescription: "400 MessageRejected",
			Matcher: func(req *engine.ParsedRequest) bool {
				if req.Operation != "SendEmail" {
					return false
				}
				to := req.FieldString("Destination.ToAddresses[0]")
				return !allowed[to]
			},
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteError(w, req.Service, http.StatusBadRequest, "MessageRejected",
					"Email address is not verified. The following identities failed the check in region US-EAST-1")
			},
		},
	}, nil
}

func presetIntermittentFailures(_ json.RawMessage) ([]*engine.Scenario, error) {
	return []*engine.Scenario{
		{
			Name:                "intermittent_failures: 20% of SendEmail → 500",
			MatcherDescription:  "operation=SendEmail AND probability=0.2",
			ResponseDescription: "500 InternalServiceError",
			Matcher: engine.All(
				engine.OperationIs("SendEmail"),
				engine.Probability(0.2),
			),
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.WriteError(w, req.Service, http.StatusInternalServerError, "InternalServiceError", "Internal service error")
			},
		},
	}, nil
}

func presetSlowResponses(_ json.RawMessage) ([]*engine.Scenario, error) {
	return []*engine.Scenario{
		{
			Name:                "slow_responses: all responses delayed 2s",
			MatcherDescription:  "*",
			ResponseDescription: "delay 2000ms + default",
			Delay:               2 * time.Second,
			Matcher:             func(req *engine.ParsedRequest) bool { return true },
			Responder: func(w http.ResponseWriter, req *engine.ParsedRequest) {
				engine.DefaultHandler(req.Operation)(w, req)
			},
		},
	}, nil
}
