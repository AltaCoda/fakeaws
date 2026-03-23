package engine

import (
	"net/http"
	"strings"
	"testing"
)

func TestFieldAt_NestedMap(t *testing.T) {
	pr := &ParsedRequest{
		Body: map[string]any{
			"Destination": map[string]any{
				"ToAddresses": []any{"alice@example.com", "bob@example.com"},
			},
		},
	}

	val, ok := pr.FieldAt("Destination.ToAddresses")
	if !ok {
		t.Fatal("expected to find Destination.ToAddresses")
	}
	arr, ok := val.([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("expected []any of len 2, got %T %v", val, val)
	}
}

func TestFieldAt_ArrayIndex(t *testing.T) {
	pr := &ParsedRequest{
		Body: map[string]any{
			"Items": []any{"a", "b", "c"},
		},
	}

	val, ok := pr.FieldAt("Items[1]")
	if !ok || val != "b" {
		t.Fatalf("expected 'b', got %v (ok=%v)", val, ok)
	}
}

func TestFieldAt_OutOfBounds(t *testing.T) {
	pr := &ParsedRequest{
		Body: map[string]any{
			"Items": []any{"a"},
		},
	}
	_, ok := pr.FieldAt("Items[5]")
	if ok {
		t.Fatal("expected false for out-of-bounds index")
	}
}

func TestFieldAt_Missing(t *testing.T) {
	pr := &ParsedRequest{Body: map[string]any{}}
	_, ok := pr.FieldAt("Nonexistent.Path")
	if ok {
		t.Fatal("expected false for missing path")
	}
}

func TestParseRequest_SES_JSON(t *testing.T) {
	body := `{"Destination":{"ToAddresses":["test@example.com"]},"Content":{"Simple":{"Subject":{"Data":"Hi"}}}}`
	r, _ := http.NewRequest("POST", "/v2/email/outbound-emails", strings.NewReader(body))
	r.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKIAEXAMPLE/20260323/us-west-2/ses/aws4_request, SignedHeaders=host, Signature=abc")

	pr := ParseRequest(r, "sesv2", "SendEmail")

	if pr.Service != "sesv2" || pr.Operation != "SendEmail" {
		t.Fatalf("unexpected service/op: %s/%s", pr.Service, pr.Operation)
	}
	if pr.Region != "us-west-2" {
		t.Fatalf("expected us-west-2, got %s", pr.Region)
	}
	if pr.AccessKeyID != "AKIAEXAMPLE" {
		t.Fatalf("expected AKIAEXAMPLE, got %s", pr.AccessKeyID)
	}

	val, ok := pr.FieldAt("Destination.ToAddresses[0]")
	if !ok || val != "test@example.com" {
		t.Fatalf("expected test@example.com, got %v", val)
	}
}

func TestParseRequest_STS_Form(t *testing.T) {
	body := "Action=AssumeRole&RoleArn=arn:aws:iam::123456789012:role/Test&RoleSessionName=test"
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	pr := ParseRequest(r, "sts", "AssumeRole")

	if pr.Body["Action"] != "AssumeRole" {
		t.Fatalf("expected Action=AssumeRole, got %v", pr.Body["Action"])
	}
	if pr.Body["RoleArn"] != "arn:aws:iam::123456789012:role/Test" {
		t.Fatalf("unexpected RoleArn: %v", pr.Body["RoleArn"])
	}
}
