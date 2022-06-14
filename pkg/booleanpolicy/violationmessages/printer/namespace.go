package printer

import (
	"github.com/stackrox/stackrox/pkg/search"
)

const (
	namespaceTemplate = `Namespace has name '{{.Namespace}}'`
)

func namespacePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		Namespace string
	}
	r := resultFields{}
	var err error
	if r.Namespace, err = getSingleValueFromFieldMap(search.Namespace.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(namespaceTemplate, r)
}
