package renderer

import (
	_ "embed"
	"text/template"

	"github.com/pkg/errors"
	helmTemplate "github.com/stackrox/rox/pkg/helm/template"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/zip"
)

//go:embed templates/public_values.yaml.tpl
var publicValuesTemplateStr string

//go:embed templates/private_values_postgres.yaml.tpl
var privateValuesYamlPostgresTemplateStr string

var (
	publicValuesTemplate = template.Must(
		helmTemplate.InitTemplate("values-public.yaml").Parse(publicValuesTemplateStr))

	privateValuesPostgresTemplate = template.Must(
		helmTemplate.InitTemplate("values-private.yaml").Parse(privateValuesYamlPostgresTemplateStr))
)

// renderNewHelmValues creates values files for the new Central Services helm charts,
// based on the given config. The values are returned as a *zip.File slice, containing
// two entries, one for `values-public.yaml`, and one for `values-private.yaml`.
func renderNewHelmValues(c Config) ([]*zip.File, error) {
	privateTemplate := privateValuesPostgresTemplate

	publicValuesBytes, err := templates.ExecuteToBytes(publicValuesTemplate, &c)
	if err != nil {
		return nil, errors.Wrap(err, "executing public values template")
	}
	privateValuesBytes, err := templates.ExecuteToBytes(privateTemplate, &c)
	if err != nil {
		return nil, errors.Wrap(err, "executing private values template")
	}

	files := []*zip.File{
		zip.NewFile("values-public.yaml", publicValuesBytes, 0),
		zip.NewFile("values-private.yaml", privateValuesBytes, zip.Sensitive),
	}
	return files, nil
}
