package auth

import (
	"net/http"
	"strings"
)

func GetAuthToken(r *http.Request) string {
	return strings.Replace(r.Header.Get("Authorization"), "Bearer ", "", 1)
}