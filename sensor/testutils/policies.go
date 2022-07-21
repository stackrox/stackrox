package testutils

import (
	"bytes"
	"fmt"
	"os"
	"unicode"

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
	for _, p := range policiesMsg.Policies {
		for _, s := range p.GetPolicySections() {
			for _, g := range s.GetPolicyGroups() {
				addSpace := func(s string) string {
					buf := &bytes.Buffer{}
					for i, r := range s {
						if unicode.IsUpper(r) && i > 0 {
							if unicode.IsLetter(rune(s[i-1])) && (!unicode.IsUpper(rune(s[i-1])) || (i < len(s)-1 && !unicode.IsUpper(rune(s[i+1])))) {
								if _, err := buf.WriteRune(' '); err != nil {
									errorList.AddError(err)
									continue
								}
							}
						}
						if _, err := buf.WriteRune(r); err != nil {
							errorList.AddError(err)
							continue
						}
					}
					return buf.String()
				}
				if g.GetFieldName() == "AppArmorProfile" {
					g.FieldName = "AppArmor Profile"
					continue
				}
				g.FieldName = addSpace(g.GetFieldName())
			}
		}
		policies = append(policies, p)
	}
	return policies, nil
}
