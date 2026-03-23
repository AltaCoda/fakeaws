package engine

import (
	"strings"
)

// Route maps an HTTP method + path pattern to an operation name.
type route struct {
	Method    string
	Pattern   string // e.g. "/v2/email/configuration-sets/{ConfigurationSetName}"
	Operation string
}

// routeMatch is the result of matching a request against the route table.
type routeMatch struct {
	Service    string
	Operation  string
	PathParams map[string]string
}

// Router resolves incoming requests to service + operation + path params.
type Router struct {
	sesRoutes []route
	stsActions map[string]string
}

// NewRouter creates a Router with the default SES v2 and STS route tables.
func NewRouter() *Router {
	return &Router{
		sesRoutes:  sesV2Routes,
		stsActions: stsActions,
	}
}

// Resolve determines the service, operation, and path params for a request.
// Returns nil if the request doesn't match any known route.
// For paths that belong to a known service prefix but don't match a specific route,
// Service will be set but Operation will be empty.
func (rt *Router) Resolve(method, path string, bodyReader func() map[string]any) *routeMatch {
	// Control plane requests are handled separately
	if strings.HasPrefix(path, "/_control/") {
		return &routeMatch{Service: "control", Operation: "control"}
	}

	// SES v2: REST-JSON, path-based routing
	if strings.HasPrefix(path, "/v2/email/") {
		for _, route := range rt.sesRoutes {
			if route.Method != method {
				continue
			}
			if params, ok := matchPattern(route.Pattern, path); ok {
				return &routeMatch{
					Service:    "sesv2",
					Operation:  route.Operation,
					PathParams: params,
				}
			}
		}
		// Known SES prefix but no matching route
		return &routeMatch{Service: "sesv2", Operation: "", PathParams: map[string]string{}}
	}

	// STS: POST to / with Action= in body
	if method == "POST" {
		body := bodyReader()
		if action, ok := body["Action"].(string); ok {
			if op, exists := rt.stsActions[action]; exists {
				return &routeMatch{
					Service:    "sts",
					Operation:  op,
					PathParams: map[string]string{},
				}
			}
		}
	}

	return nil
}

// matchPattern matches a URL path against a route pattern with {param} placeholders.
// Returns extracted parameters and whether the match succeeded.
func matchPattern(pattern, path string) (map[string]string, bool) {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)
	for i, pp := range patternParts {
		if strings.HasPrefix(pp, "{") && strings.HasSuffix(pp, "}") {
			paramName := pp[1 : len(pp)-1]
			params[paramName] = pathParts[i]
		} else if pp != pathParts[i] {
			return nil, false
		}
	}

	return params, true
}
