package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/search"
)

const (
	replicasTemplate = `{{- if eq .Replicas 1}}{{.Replicas}} replica is{{else}}{{.Replicas}} replicas are{{end}} defined.`
)

func replicasPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		Replicas int64
	}

	r := resultFields{}
	var err error
	replicas, err := getSingleValueFromFieldMap(search.Replicas.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.Replicas, err = strconv.ParseInt(replicas, 10, 64); err != nil {
		return nil, err
	}
	return executeTemplate(replicasTemplate, r)
}
