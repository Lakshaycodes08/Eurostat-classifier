package util

import "os"

// GetEnvRequired returns the value of the given environment variable
// and a boolean indicating whether it was set.
//
// The kernel maps missing/empty required env vars to the auth error
// exit code (3). Thin clients must not attempt to "help" here.
func GetEnvRequired(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}

