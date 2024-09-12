package images

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Overrides defines a mapping from image override environment variable settings to
// Helm chart configuration paths.
type Overrides map[env.Setting]string

// ToValues returns a Helm chart values object that applies the override settings that are set.
func (o Overrides) ToValues() (chartutil.Values, error) {
	vals := chartutil.Values{}
	for setting, configPath := range o {
		val := setting.Setting()
		if val == "" {
			continue
		}
		newVals, err := helmUtil.ValuesForKVPair(configPath, val)
		if err != nil {
			return nil, errors.Wrapf(err, "applying image override from %s", setting.EnvVar())
		}
		vals = chartutil.CoalesceTables(vals, newVals)
	}
	return vals, nil
}
