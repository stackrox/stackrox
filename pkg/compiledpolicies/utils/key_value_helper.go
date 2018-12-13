package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// MatchRequiredMap returns violations for the key values passed based on the input regex's.
func MatchRequiredMap(m map[string]string, key, value *regexp.Regexp, name string) []*storage.Alert_Violation {
	for k, v := range m {
		if key != nil && value != nil {
			if key.MatchString(k) && value.MatchString(v) {
				return nil
			}
		} else if key != nil {
			if key.MatchString(k) {
				return nil
			}
		} else if value != nil {
			if value.MatchString(v) {
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
	return []*storage.Alert_Violation{
		{
			Message: fmt.Sprintf("Could not find %s that matched required %s policy (%s)", name, name, strings.Join(fields, ",")),
		},
	}
}
