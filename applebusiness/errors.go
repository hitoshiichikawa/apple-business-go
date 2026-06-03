package applebusiness

import "errors"

// 型付きエラー判定。いずれも err（またはラップされたもの）が *APIError で、
// 対応する HTTP ステータスの場合に true を返す。
//
//	if applebusiness.IsNotFound(err) { ... }

// IsNotFound は 404 を判定する。
func IsNotFound(err error) bool { return statusIs(err, 404) }

// IsRateLimited は 429 を判定する（リトライ枯渇時の戻り値も含む）。
func IsRateLimited(err error) bool { return statusIs(err, 429) }

// IsUnauthorized は 401 を判定する。
func IsUnauthorized(err error) bool { return statusIs(err, 401) }

// IsForbidden は 403 を判定する（relationships の権限不足などで発生し得る）。
func IsForbidden(err error) bool { return statusIs(err, 403) }

// IsConflict は 409 を判定する。
func IsConflict(err error) bool { return statusIs(err, 409) }

func statusIs(err error, code int) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == code
	}
	return false
}
