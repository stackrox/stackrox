package settingswatch

import (
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
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
	if err := proto.Unmarshal(runTimePoliciesData, &policyList); err != nil {
		return nil, errors.Wrap(err, "unmarshaling decompressed policies data")
	}
	return &policyList, nil
}
