package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParsedRequest is the normalized representation of an incoming AWS request.
// Matchers and responders consume this rather than raw *http.Request.
type ParsedRequest struct {
	Service   string // "sesv2", "sts"
	Operation string
	Timestamp time.Time

	// Raw HTTP
	Method  string
	Path    string
	Headers http.Header
	RawBody []byte

	// Parsed payload — JSON body for SES, form values for STS
	Body map[string]any

	// AWS metadata extracted from SigV4 Authorization header
	Region      string
	AccessKeyID string

	// Path parameters extracted by the router (e.g. "Name" from /v2/email/configuration-sets/{Name})
	PathParams map[string]string

	// Context from the original HTTP request. Use for timeout-aware responders.
	Context context.Context
}

var sigv4Re = regexp.MustCompile(`Credential=([^/]+)/\d{8}/([^/]+)/`)

// ParseRequest creates a ParsedRequest from a raw HTTP request.
func ParseRequest(r *http.Request, service, operation string) *ParsedRequest {
	body, _ := io.ReadAll(r.Body)

	pr := &ParsedRequest{
		Service:    service,
		Operation:  operation,
		Timestamp:  time.Now(),
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    r.Header,
		RawBody:    body,
		Body:       make(map[string]any),
		PathParams: make(map[string]string),
		Context:    r.Context(),
	}

	// Extract Region and AccessKeyID from SigV4 Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		if matches := sigv4Re.FindStringSubmatch(auth); len(matches) == 3 {
			pr.AccessKeyID = matches[1]
			pr.Region = matches[2]
		}
	}

	// Parse body based on service
	switch service {
	case "sesv2":
		if len(body) > 0 {
			_ = json.Unmarshal(body, &pr.Body)
		}
	case "sts":
		// STS uses application/x-www-form-urlencoded
		if len(body) > 0 {
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			_ = r.ParseForm()
			for k, v := range r.PostForm {
				if len(v) == 1 {
					pr.Body[k] = v[0]
				} else {
					pr.Body[k] = v
				}
			}
		}
	}

	return pr
}

// FieldAt traverses the Body using dot-path notation.
// Supports nested maps and array indexing: "Destination.ToAddresses[0]"
func (pr *ParsedRequest) FieldAt(path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = pr.Body

	for _, part := range parts {
		name, idx, hasIdx := parseArrayIndex(part)

		switch m := current.(type) {
		case map[string]any:
			val, ok := m[name]
			if !ok {
				return nil, false
			}
			if hasIdx {
				current = val
				current, ok = indexInto(current, idx)
				if !ok {
					return nil, false
				}
			} else {
				current = val
			}
		default:
			return nil, false
		}
	}

	return current, true
}

// parseArrayIndex parses "foo[2]" into ("foo", 2, true) or "foo" into ("foo", 0, false).
func parseArrayIndex(part string) (string, int, bool) {
	bracketIdx := strings.Index(part, "[")
	if bracketIdx == -1 {
		return part, 0, false
	}
	if !strings.HasSuffix(part, "]") || len(part) < bracketIdx+3 {
		return part, 0, false
	}
	name := part[:bracketIdx]
	idxStr := part[bracketIdx+1 : len(part)-1]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return part, 0, false
	}
	return name, idx, true
}

func indexInto(val any, idx int) (any, bool) {
	switch arr := val.(type) {
	case []any:
		if idx < 0 || idx >= len(arr) {
			return nil, false
		}
		return arr[idx], true
	case []string:
		if idx < 0 || idx >= len(arr) {
			return nil, false
		}
		return arr[idx], true
	default:
		return nil, false
	}
}

// FieldString is a convenience wrapper returning the field as a string.
func (pr *ParsedRequest) FieldString(path string) string {
	val, ok := pr.FieldAt(path)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", val)
}
