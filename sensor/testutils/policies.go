package testutils

import (
	"fmt"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	localSensor "github.com/stackrox/rox/generated/tools/local-sensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// GetPoliciesFromFile reads a file containing storage.Policy. Return a slice of storage.Policy with the content of the file
func GetPoliciesFromFile(fileName string) (policies []*storage.Policy, retError error) {
	policiesMsg := &localSensor.LocalSensorPolicies{}
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
	if err := jsonpb.Unmarshal(file, policiesMsg); err != nil {
		errorList.AddStringf("error unmarshaling %s: %s\n", fileName, err)
		return
	}
	policies = append(policies, policiesMsg.Policies...)
	return policies, nil
}
