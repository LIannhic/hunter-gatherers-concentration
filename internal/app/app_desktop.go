//go:build !js

package app

func isWasm() bool {
	return false
}
