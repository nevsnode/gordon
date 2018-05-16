package utils

import (
	"net/http"
	"strings"
)

// PrepareHTTPHeader prepares header-keys to be used by the go http-client
func PrepareHTTPHeader(hKey string) string {
	hKey = http.CanonicalHeaderKey(strings.Replace(hKey, "_", "-", -1))
	return hKey
}
