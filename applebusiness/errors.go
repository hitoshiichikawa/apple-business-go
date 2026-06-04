package applebusiness

import "errors"

// Typed error predicates. Each returns true when err (or a wrapped error) is an
// *APIError with the corresponding HTTP status.
//
//	if applebusiness.IsNotFound(err) { ... }

// IsNotFound reports whether err is a 404.
func IsNotFound(err error) bool { return statusIs(err, 404) }

// IsRateLimited reports whether err is a 429 (including the value returned when retries are exhausted).
func IsRateLimited(err error) bool { return statusIs(err, 429) }

// IsUnauthorized reports whether err is a 401.
func IsUnauthorized(err error) bool { return statusIs(err, 401) }

// IsForbidden reports whether err is a 403 (which can occur, e.g., on insufficient relationship permissions).
func IsForbidden(err error) bool { return statusIs(err, 403) }

// IsConflict reports whether err is a 409.
func IsConflict(err error) bool { return statusIs(err, 409) }

func statusIs(err error, code int) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == code
	}
	return false
}
