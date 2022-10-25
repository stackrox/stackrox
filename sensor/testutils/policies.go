package testutils

import (
	"fmt"
	"io/ioutil"

	"github.com/stackrox/rox/generated/storage"
	localSensor "github.com/stackrox/rox/generated/tools/local-sensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"google.golang.org/protobuf/encoding/protojson"
)

// GetPoliciesFromFile reads a file containing storage.Policy. Return a slice of storage.Policy with the content of the file
func GetPoliciesFromFile(fileName string) (policies []*storage.Policy, retError error) {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		retError = fmt.Errorf("error opening %s: %w\n", fileName, err)
		return
	}
	errorList := errorhelpers.NewErrorList("read policies from file")
	var policiesMsg localSensor.LocalSensorPolicies
	if err := protojson.Unmarshal(fileContents, &policiesMsg); err != nil {
		errorList.AddStringf("error unmarshaling %s: %s\n", fileName, err)
		return
	}
	policies = append(policies, policiesMsg.Policies...)
	return policies, nil
}
