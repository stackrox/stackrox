package printer

import (
	"github.com/stackrox/rox/pkg/search"
)

const (
	serviceAccountTemplate = `Service Account is set to '{{.ServiceAccount}}'`
)

func serviceAccountPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ServiceAccount string
	}

	serviceAccount, err := getSingleValueFromFieldMap(search.ServiceAccountName.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	return executeTemplate(serviceAccountTemplate, &resultFields{ServiceAccount: serviceAccount})
}
