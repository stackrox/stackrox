package helmutil

import (
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
)

// ValuesForKVPair returns a chartutil.Values that has exactly the single key identified by `configPath` set to
// `val`. The `chartutil.CoalesceTables` function can be used to merge this into an existing values
// dictionary.
func ValuesForKVPair(configPath string, val interface{}) (chartutil.Values, error) {
	if configPath == "" {
		return nil, errors.New("empty config path")
	}
	keyPath := strings.Split(configPath, ".")

	mapForSet := map[string]interface{}{keyPath[len(keyPath)-1]: val}
	for i := len(keyPath) - 2; i >= 0; i-- {
		mapForSet = map[string]interface{}{keyPath[i]: mapForSet}
	}
	return chartutil.Values(mapForSet), nil
}
