package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// MatchRequiredKeyValue returns violations for the key values passed based on the input regex's.
func MatchRequiredKeyValue(deploymentKeyValues []*v1.Deployment_KeyValue, key, value *regexp.Regexp, name string) []*v1.Alert_Violation {
	for _, keyValue := range deploymentKeyValues {
		if key != nil && value != nil {
			if key.MatchString(keyValue.GetKey()) && value.MatchString(keyValue.GetValue()) {
				return nil
			}
		} else if key != nil {
			if key.MatchString(keyValue.GetKey()) {
				return nil
			}
		} else if value != nil {
			if value.MatchString(keyValue.GetValue()) {
				return nil
			}
		}
	}
	var fields []string
	if key != nil {
		fields = append(fields, fmt.Sprintf("key='%s'", key))
	}
	if value != nil {
		fields = append(fields, fmt.Sprintf("value='%s'", value))
	}
	return []*v1.Alert_Violation{
		{
			Message: fmt.Sprintf("Could not find %s that matched required %s policy (%s)", name, name, strings.Join(fields, ",")),
		},
	}
}
