package httputil

import "net/http"

// SetCookie works like `http.SetCookie`, but operates on an `http.Header` (which must be a response header) instead of
// an `http.ResponseWriter`. The behavior is excactly the same, including silently dropping any errors in case the
// cookie could not be encoded.
func SetCookie(responseHdr http.Header, cookie *http.Cookie) {
	if encoded := cookie.String(); encoded != "" {
		responseHdr.Add("Set-Cookie", encoded)
	}
}
