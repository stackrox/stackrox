package secrets

import "strings"

// secretKeys lists the keys that have secret values so we should scrub the keys out of the map before returning from the api
var secretKeys = map[string]struct{}{
	"password":  {},
	"token":     {},
	"secretKey": {},
}

// ScrubSecrets removes secret keys from a map[string]string and returns a new copy without secrets
func ScrubSecrets(m map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range m {
		if _, isSecret := secretKeys[strings.ToLower(k)]; !isSecret {
			newMap[k] = v
		}
	}
	return newMap
}
