package engine

import "fmt"

// OperationIs matches requests for a specific operation name.
func OperationIs(op string) Matcher {
	return func(req *ParsedRequest) bool {
		return req.Operation == op
	}
}

// ServiceIs matches requests for a specific service.
func ServiceIs(svc string) Matcher {
	return func(req *ParsedRequest) bool {
		return req.Service == svc
	}
}

// FieldEquals matches when a dot-path field equals the given string value.
func FieldEquals(path, value string) Matcher {
	return func(req *ParsedRequest) bool {
		v, ok := req.FieldAt(path)
		if !ok {
			return false
		}
		return fmt.Sprintf("%v", v) == value
	}
}

// HeaderEquals matches when a header has the given value.
func HeaderEquals(key, value string) Matcher {
	return func(req *ParsedRequest) bool {
		return req.Headers.Get(key) == value
	}
}

// All returns a matcher that requires all sub-matchers to pass (AND).
func All(matchers ...Matcher) Matcher {
	return func(req *ParsedRequest) bool {
		for _, m := range matchers {
			if !m(req) {
				return false
			}
		}
		return true
	}
}

// Any returns a matcher that requires at least one sub-matcher to pass (OR).
func Any(matchers ...Matcher) Matcher {
	return func(req *ParsedRequest) bool {
		for _, m := range matchers {
			if m(req) {
				return true
			}
		}
		return false
	}
}
