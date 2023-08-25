package testutils

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/generated/storage"
	localSensor "github.com/stackrox/rox/generated/tools/local-sensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/jsonutil"
)

// GetPoliciesFromFile reads a file containing storage.Policy. Return a slice of storage.Policy with the content of the file
func GetPoliciesFromFile(fileName string) (policies []*storage.Policy, retError error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		retError = fmt.Errorf("error opening %s: %w\n", fileName, err)
		return
	}
	errorList := errorhelpers.NewErrorList("read policies from file")
	defer func() {
		if err = file.Close(); err != nil {
			errorList.AddError(err)
		}
		retError = errorList.ToError()
	}()
	var policiesMsg localSensor.LocalSensorPolicies
	if err := jsonutil.JSONReaderToProto(file, &policiesMsg); err != nil {
		errorList.AddStringf("error unmarshaling %s: %s\n", fileName, err)
		return
	}
	policies = append(policies, policiesMsg.Policies...)
	return policies, nil
}
