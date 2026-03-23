package engine

import (
	"encoding/json"
	"net/http"
	"strings"
)

// writePermissiveResponse returns a valid-looking response for AWS services
// that fakeaws doesn't explicitly implement. This allows preflight checks
// and other non-SES/STS SDK calls to succeed when AWS_ENDPOINT_URL is set.
func writePermissiveResponse(w http.ResponseWriter, r *http.Request) {
	// AWS SDK sends the target service+operation in the X-Amz-Target header
	// (e.g., "AWSEvents.DescribeRule") or via the Authorization header's
	// service component.
	target := r.Header.Get("X-Amz-Target")

	if strings.HasPrefix(target, "AWSEvents.") {
		writeEventBridgeStub(w, r, target)
		return
	}

	// Generic fallback: empty JSON 200
	WriteJSONResponse(w, http.StatusOK, map[string]any{})
}

func writeEventBridgeStub(w http.ResponseWriter, r *http.Request, target string) {
	op := strings.TrimPrefix(target, "AWSEvents.")

	var body any
	switch op {
	case "DescribeRule":
		// Read the request to extract the rule name
		var input map[string]any
		json.NewDecoder(r.Body).Decode(&input)
		name, _ := input["Name"].(string)
		body = map[string]any{
			"Name":  name,
			"State": "ENABLED",
			"Arn":   GenerateARN("events", "rule/default/"+name),
		}
	case "ListTargetsByRule":
		body = map[string]any{
			"Targets": []map[string]any{
				{
					"Id":  "sendops-webhook",
					"Arn": GenerateARN("execute-api", "api/sendops-webhook"),
				},
			},
		}
	default:
		body = map[string]any{}
	}

	WriteJSONResponse(w, http.StatusOK, body)
}
