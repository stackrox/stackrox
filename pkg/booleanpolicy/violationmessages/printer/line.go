package printer

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

var (
	maxDockerfileLineLength = 32
)

const (
	lineTemplate = `Dockerfile line '{{.Instruction}} {{.Line}}' found{{- if .ContainerName}} in container '{{.ContainerName}}'{{end}}`
)

func linePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Instruction   string
		Line          string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	value, err := getSingleValueFromFieldMap(augmentedobjs.DockerfileLineCustomTag, fieldMap)
	if err != nil {
		return nil, errors.New("invalid dockerfile line in result")
	}
	dockerfileLine := strings.SplitN(value, augmentedobjs.CompositeFieldCharSep, 2)
	if len(dockerfileLine) != 2 {
		return nil, errors.New("failed to parse docker file line result")
	}
	r.Instruction = dockerfileLine[0]
	r.Line = dockerfileLine[1]
	if len(r.Line) > maxDockerfileLineLength {
		r.Line = r.Line[:maxDockerfileLineLength] + "..."
	}
	return executeTemplate(lineTemplate, r)
}
