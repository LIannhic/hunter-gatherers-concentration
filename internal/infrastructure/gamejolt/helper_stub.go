//go:build !js

package gamejolt

// GetCredentialsFromURL est un stub pour les builds non-WASM.
func GetCredentialsFromURL() (string, string) {
	return "", ""
}
