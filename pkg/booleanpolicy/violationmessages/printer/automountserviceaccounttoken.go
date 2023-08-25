package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/search"
)

const (
	//#nosec G101 -- This is a false positive
	automountServiceAccountTokenTemplate = `Deployment {{- if .AutomountServiceAccountToken }} mounts{{else}} does not mount{{end}} the service account tokens.`
)

func automountServiceAccountTokenPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		AutomountServiceAccountToken bool
	}

	r := resultFields{}
	var err error
	automountServiceAccountToken, err := getSingleValueFromFieldMap(search.AutomountServiceAccountToken.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.AutomountServiceAccountToken, err = strconv.ParseBool(automountServiceAccountToken); err != nil {
		return nil, err
	}
	return executeTemplate(automountServiceAccountTokenTemplate, r)
}
