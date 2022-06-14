package printer

import (
	"strconv"

	"github.com/stackrox/stackrox/pkg/search"
)

const (
	livenessProbeDefinedTemplate  = `Liveness probe is{{- if not .LivenessProbeDefined}} not{{end}} defined for container '{{.ContainerName}}'`
	readinessProbeDefinedTemplate = `Readiness probe is{{- if not .ReadinessProbeDefined}} not{{end}} defined for container '{{.ContainerName}}'`
)

func livenessProbeDefinedPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName        string
		LivenessProbeDefined bool
	}

	r := resultFields{}
	var err error
	if r.ContainerName, err = getSingleValueFromFieldMap(search.ContainerName.String(), fieldMap); err != nil {
		return nil, err
	}
	livenessProbeDefined, err := getSingleValueFromFieldMap(search.LivenessProbeDefined.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.LivenessProbeDefined, err = strconv.ParseBool(livenessProbeDefined); err != nil {
		return nil, err
	}
	return executeTemplate(livenessProbeDefinedTemplate, r)
}

func readinessProbeDefinedPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName         string
		ReadinessProbeDefined bool
	}

	r := resultFields{}
	var err error
	if r.ContainerName, err = getSingleValueFromFieldMap(search.ContainerName.String(), fieldMap); err != nil {
		return nil, err
	}
	readinessProbeDefined, err := getSingleValueFromFieldMap(search.ReadinessProbeDefined.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.ReadinessProbeDefined, err = strconv.ParseBool(readinessProbeDefined); err != nil {
		return nil, err
	}
	return executeTemplate(readinessProbeDefinedTemplate, r)
}
