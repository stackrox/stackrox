package testutils

import (
	"errors"
	"fmt"
	"os"

	"github.com/stackrox/rox/generated/storage"
	localSensor "github.com/stackrox/rox/generated/tools/local-sensor"
	"github.com/stackrox/rox/pkg/jsonutil"
)

// GetPoliciesFromFile reads a file containing storage.Policy. Return a slice of storage.Policy with the content of the file
func GetPoliciesFromFile(fileName string) (policies []*storage.Policy, retError error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		retError = fmt.Errorf("error opening %s: %w\n", fileName, err)
		return
	}
	var readErrs error
	defer func() {
		if err = file.Close(); err != nil {
			readErrs = errors.Join(readErrs, err)
		}
		retError = readErrs
	}()
	var policiesMsg localSensor.LocalSensorPolicies
	if err := jsonutil.JSONReaderToProto(file, &policiesMsg); err != nil {
		readErrs = errors.Join(readErrs, fmt.Errorf("unmarshaling %s: %w", fileName, err))
		return
	}
	policies = append(policies, policiesMsg.Policies...)
	return policies, nil
}
