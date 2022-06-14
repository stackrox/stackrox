package printer

import (
	"strconv"

	"github.com/stackrox/stackrox/pkg/search"
)

const (
	replicasTemplate = `Replicas is set to '{{.Replicas}}'`
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
