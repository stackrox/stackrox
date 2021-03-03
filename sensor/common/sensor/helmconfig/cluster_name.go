package helmconfig

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ErrClusterNameDoesNotMatch indicates desired and effective cluster name diverged.
var ErrClusterNameDoesNotMatch = errors.New("cluster name does not match")

const errClusterNameDoesNotMatchMsg = `It looks like you have changed the name of the cluster in your Helm configuration from %s to %s.
Note that this will NOT rename the cluster in StackRox, but instead create a new cluster with name %s, leaving cluster %s inactive (see the chart README for more details).
If this is what you want, set the "confirmNewClusterName" property to %q.`

// CheckEffectiveClusterName validates if the effective cluster name matches with the applied cluster name
func CheckEffectiveClusterName(helmConfig *central.HelmManagedConfigInit) error {
	if helmConfig.GetClusterName() == "" {
		return nil
	}

	oldName, err := getEffectiveClusterName()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debugf("reading effective cluster name failed: %s", err)
			return nil
		}
		return errors.Wrap(err, "reading effective cluster oldName")
	}

	if oldName != helmConfig.ClusterName {
		return errors.Wrapf(ErrClusterNameDoesNotMatch, errClusterNameDoesNotMatchMsg, oldName,
			helmConfig.GetClusterName(), helmConfig.GetClusterName(), oldName, helmConfig.GetClusterName())
	}
	return nil
}
