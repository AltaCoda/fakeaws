package engine

import "testing"

func TestRouter_SES_SendEmail(t *testing.T) {
	rt := NewRouter()
	m := rt.Resolve("POST", "/v2/email/outbound-emails", emptyBody)
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Service != "sesv2" || m.Operation != "SendEmail" {
		t.Fatalf("got %s/%s", m.Service, m.Operation)
	}
}

func TestRouter_SES_PathParams(t *testing.T) {
	rt := NewRouter()
	m := rt.Resolve("GET", "/v2/email/configuration-sets/MySet", emptyBody)
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Operation != "GetConfigurationSet" {
		t.Fatalf("got operation %s", m.Operation)
	}
	if m.PathParams["ConfigurationSetName"] != "MySet" {
		t.Fatalf("got param %s", m.PathParams["ConfigurationSetName"])
	}
}

func TestRouter_STS_AssumeRole(t *testing.T) {
	rt := NewRouter()
	m := rt.Resolve("POST", "/", func() map[string]any {
		return map[string]any{"Action": "AssumeRole"}
	})
	if m == nil {
		t.Fatal("expected match")
	}
	if m.Service != "sts" || m.Operation != "AssumeRole" {
		t.Fatalf("got %s/%s", m.Service, m.Operation)
	}
}

func TestRouter_UnknownPath(t *testing.T) {
	rt := NewRouter()
	m := rt.Resolve("GET", "/unknown/path", emptyBody)
	if m != nil {
		t.Fatalf("expected nil, got %+v", m)
	}
}

func TestRouter_ControlPlane(t *testing.T) {
	rt := NewRouter()
	m := rt.Resolve("GET", "/_control/scenarios", emptyBody)
	if m == nil || m.Service != "control" {
		t.Fatal("expected control plane match")
	}
}

func emptyBody() map[string]any { return map[string]any{} }
