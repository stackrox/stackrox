package secrets

import "strings"

// secretKeys lists keys that have secret values that should be scrubbed
// out of a config before returning it in the API.
type secretKeys map[string]struct{}

func newSecretKeys(keys []string) secretKeys {
	sk := make(secretKeys)
	for _, k := range keys {
		sk[strings.ToLower(k)] = struct{}{}
	}
	return sk
}

func (sk secretKeys) shouldScrub(key string) bool {
	_, present := sk[strings.ToLower(key)]
	return present
}

var scrubber = newSecretKeys([]string{
	"oauthToken",
	"password",
	"secretKey",
	"serviceAccount",
})

// ScrubSecrets removes secret keys from a map[string]string and returns a new copy without secrets.
func ScrubSecrets(m map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range m {
		if !scrubber.shouldScrub(k) {
			newMap[k] = v
		}
	}
	return newMap
}
