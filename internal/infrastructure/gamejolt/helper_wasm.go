//go:build js

package gamejolt

import (
	"net/url"
	"syscall/js"
)

// GetCredentialsFromURL tente d'extraire le username et le token Game Jolt
// depuis les paramètres de l'URL de la page (gjapi_username et gjapi_token).
func GetCredentialsFromURL() (string, string) {
	window := js.Global().Get("window")
	if window.IsUndefined() {
		return "", ""
	}
	location := window.Get("location")
	if location.IsUndefined() {
		return "", ""
	}
	search := location.Get("search").String()
	if search == "" || len(search) < 2 {
		return "", ""
	}

	// search commence par '?'
	u, err := url.ParseQuery(search[1:])
	if err != nil {
		return "", ""
	}

	return u.Get("gjapi_username"), u.Get("gjapi_token")
}
