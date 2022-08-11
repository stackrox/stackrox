package printer

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	sbomTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' image` +
		`{{else}}Image{{end}} has {{if .Unverified}}no SBOM` +
		`{{else}}a SBOM that only partially covers the contents of the image{{end}}`
)

func sbomPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Unverified    bool
	}

	r := resultFields{
		ContainerName: maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap),
		Unverified:    true,
	}

	status, err := getSingleValueFromFieldMap(search.SBOMVerificationStatus.String(), fieldMap)
	if err != nil {
		return nil, err
	}

	if status == strings.ToLower(storage.SBOMVerificationResult_PARTIALLY_COVERED.String()) {
		r.Unverified = false
	}

	return executeTemplate(sbomTemplate, r)
}
