package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/search"
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
	replicasFloat, err := strconv.ParseFloat(replicas, 64)
	if err != nil {
		return nil, err
	}
	r.Replicas = int64(replicasFloat)
	return executeTemplate(replicasTemplate, r)
}
