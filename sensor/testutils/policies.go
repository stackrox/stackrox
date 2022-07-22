package testutils

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	localSensor "github.com/stackrox/rox/generated/tools/local-sensor"
	"github.com/stackrox/rox/pkg/booleanpolicy"
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
	policyMap, err := getPolicyFieldMap()
	if err != nil {
		retError = err
		return
	}
	for _, p := range policiesMsg.Policies {
		for _, s := range p.GetPolicySections() {
			for _, g := range s.GetPolicyGroups() {
				if strings.Contains(g.GetFieldName(), " ") {
					continue
				}
				// if the unmarshaling removes the spaces we need to get the correct FieldName
				fieldName, ok := policyMap[g.GetFieldName()]
				if !ok {
					errorList.AddStringf("policy field %s not found", g.GetFieldName())
					continue
				}
				g.FieldName = fieldName
			}
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func getPolicyFieldMap() (map[string]string, error) {
	ret := make(map[string]string)
	f := booleanpolicy.FieldMetadataSingleton()
	err := f.ForEachFieldMetadata(func(fieldName string, m *booleanpolicy.MetadataAndQB) error {
		ret[strings.ReplaceAll(fieldName, " ", "")] = fieldName
		return nil
	})
	return ret, err
}
