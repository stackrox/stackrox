package helmtpl

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const (
	// IgnoreFile is the file name of the file containing ignore rules for helm chart templates.
	IgnoreFile = ".helmtplignore"

	// TemplateFileSuffix is the file suffix for files that should be rendered as templates
	// when instantiating.
	TemplateFileSuffix = ".htpl"
	// NoTemplateFileSuffix is a suffix that allows marking a file explicitly as not a template.
	// This isn't generally needed, but provides an escape hatch to make sure that the
	// file extension-based enabling of templating semantics does not constrain the set of valid
	// file names, as any non-template file that (for whatever reason) would require the .htpl
	// extension can simply be named `foo.htpl.hnotpl` to result in a `foo.htpl` file obtained
	// without any templating logic.
	NoTemplateFileSuffix = ".hnotpl"
)

var (
	filterOpts = helmutil.FilterOptions{
		IgnoreFileName:          IgnoreFile,
		ApplyDefaultIgnoreRules: false,
		KeepIgnoreFile:          false,
	}
)

type element struct {
	name string
	get  func(vals interface{}) ([]byte, error)
}

// ChartTemplate is a template for a Helm chart. It can be instantiated from a meta-values
// structure, and loaded directly as a helm chart.
type ChartTemplate struct {
	elements []element
}

// Load loads a chart template from a set of files. If a file named `.helmtplignore` is
// part of the specified files, it is parsed as an ignorefile with the same syntax (and rule
// semantics) as the .helmignore files.
func Load(files []*loader.BufferedFile) (*ChartTemplate, error) {
	filtered, err := helmutil.FilterFilesWithOptions(files, filterOpts)
	if err != nil {
		return nil, errors.Wrap(err, "filtering helmtpl files")
	}

	elems := make([]element, 0, len(filtered))
	for _, file := range filtered {
		elem := element{
			name: file.Name,
		}
		data := file.Data

		if stringutils.ConsumeSuffix(&elem.name, TemplateFileSuffix) {
			tpl, err := template.New(elem.name).Delims("[<", ">]").Funcs(sprig.TxtFuncMap()).Parse(string(data))
			if err != nil {
				return nil, errors.Wrapf(err, "parsing template file %s", file.Name)
			}
			elem.get = func(vals interface{}) ([]byte, error) {
				var keepEmpty bool
				keepEmptyFuncMap := template.FuncMap{
					"helmTplKeepEmptyFile": func() string {
						keepEmpty = true
						return ""
					},
				}
				var buf bytes.Buffer
				if err := template.Must(tpl.Clone()).Funcs(keepEmptyFuncMap).Execute(&buf, vals); err != nil {
					return nil, errors.Wrapf(err, "instantiating template file %s", tpl.Name())
				}
				renderedData := buf.Bytes()
				if !keepEmpty && len(bytes.TrimSpace(renderedData)) == 0 {
					return nil, nil
				}
				return renderedData, nil
			}
		} else {
			stringutils.ConsumeSuffix(&elem.name, NoTemplateFileSuffix)
			elem.get = func(interface{}) ([]byte, error) {
				return data, nil
			}
		}

		elems = append(elems, elem)
	}

	return &ChartTemplate{
		elements: elems,
	}, nil
}

// InstantiateRaw instantiates a chart template using the given meta-values. The result is
// a set of raw files, which can be loaded as a Helm template. Note that the resulting set of
// files might even contain a `.helmignore` file, in order to apply it before loading the
// instantiated chart, use `helmutil.LoadChart` instead of `loader.LoadFiles`.
func (t *ChartTemplate) InstantiateRaw(metaVals interface{}) ([]*loader.BufferedFile, error) {
	files := make([]*loader.BufferedFile, 0, len(t.elements))
	for _, elem := range t.elements {
		data, err := elem.get(metaVals)
		if err != nil {
			return nil, errors.Wrapf(err, "instantiating file %s", elem.name)
		}
		if data == nil {
			continue
		}
		files = append(files, &loader.BufferedFile{
			Name: elem.name,
			Data: data,
		})
	}

	// The template might include a `.helmtplignore.htpl` file, which is intended to exclude
	// files _post_ rendering, in order to realize file-level excludes based on meta values.
	// Apply this filtering here.
	return helmutil.FilterFilesWithOptions(files, filterOpts)
}

// InstantiateAndLoad instantiates a chart template using the given meta-values, and loads
// the resulting Helm chart. It is a convenience method, combining `InstantiateRaw` and
// `helmutil.LoadChart`.
func (t *ChartTemplate) InstantiateAndLoad(metaVals interface{}) (*chart.Chart, error) {
	instantiatedFiles, err := t.InstantiateRaw(metaVals)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating chart template files")
	}

	ch, err := loader.LoadFiles(instantiatedFiles)
	if err != nil {
		return nil, errors.Wrap(err, "loading instantiated chart files")
	}

	return ch, nil
}
