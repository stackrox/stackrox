package settingswatch

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/gziputil"
)

func getPoliciesFromFile(file string) (*storage.PolicyList, error) {
	dataGZ, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading policies from file %s", file)
	}

	return decompressAndUnmarshalPolicies(dataGZ)
}

func decompressAndUnmarshalPolicies(data []byte) (*storage.PolicyList, error) {
	runTimePoliciesData, err := gziputil.Decompress(data)
	if err != nil {
		return nil, errors.Wrap(err, "decompressing policies")
	}

	var policyList storage.PolicyList
	if err := policyList.UnmarshalVTUnsafe(runTimePoliciesData); err != nil {
		return nil, errors.Wrap(err, "unmarshaling decompressed policies data")
	}
	return &policyList, nil
}

func decompressAndUnmarshalClusterLabels(data []byte) (map[string]string, error) {
	if len(data) == 0 {
		return nil, nil
	}

	clusterLabelsData, err := gziputil.Decompress(data)
	if err != nil {
		return nil, errors.Wrap(err, "decompressing cluster labels")
	}

	var clusterLabels sensor.ClusterLabels
	if err := clusterLabels.UnmarshalVTUnsafe(clusterLabelsData); err != nil {
		return nil, errors.Wrap(err, "unmarshaling decompressed cluster labels data")
	}
	return clusterLabels.GetLabels(), nil
}
