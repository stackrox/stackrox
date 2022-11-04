package framework

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/helmtest/internal/schemas"

	"github.com/stackrox/helmtest/internal/compiler"
	"github.com/stackrox/helmtest/internal/logic"
	"github.com/stackrox/helmtest/internal/rox-imported/sliceutils"
	"github.com/stackrox/helmtest/internal/rox-imported/stringutils"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/util/openapi"
	"k8s.io/kubectl/pkg/util/openapi/validation"
	k8sYaml "sigs.k8s.io/yaml"
)

type runner struct {
	t    *testing.T
	test *Test
	tgt  *Target
}

func (r *runner) Assert() *assert.Assertions {
	return assert.New(r.t)
}

func (r *runner) Require() *require.Assertions {
	return require.New(r.t)
}

func (r *runner) readAndValidateYAML(fileName, fileContents string, resources openapi.Resources) []unstructured.Unstructured {
	validator := validation.NewSchemaValidation(resources)

	yamlReader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(fileContents)))

	var objs []unstructured.Unstructured

	docCounter := 0
	var yamlDoc []byte
	var err error
	var emptyDocs []int
	for yamlDoc, err = yamlReader.Read(); err == nil; yamlDoc, err = yamlReader.Read() {
		docCounter++

		// We can tolerate empty documents in some circumstances (such as when the entire file is empty), but having
		// empty documents in an overall non-empty file will at least cause lint errors.
		if len(bytes.TrimSpace(yamlDoc)) == 0 {
			emptyDocs = append(emptyDocs, docCounter)
			continue
		}

		// Do the validation before converting to JSON such that we get accurate line numbers.
		validationErr := validator.ValidateBytes(yamlDoc)
		r.Assert().NoErrorf(validationErr, "YAML document #%d in file %s failed validation", docCounter, fileName)

		// YAMLToJSONStrict will not only convert to YAML, but also validate that there are no duplicate keys.
		jsonBytes, err := k8sYaml.YAMLToJSONStrict(yamlDoc)
		if !r.Assert().NoErrorf(err, "could not convert YAML document #%d in file %s to JSON", docCounter, fileName) {
			continue
		}

		obj, _, err := unstructured.UnstructuredJSONScheme.Decode(jsonBytes, nil, nil)
		if !r.Assert().NoErrorf(err, "could not decode Kubernetes object in YAML document #%d in file %s", docCounter, fileName) {
			continue
		}

		unstructuredObj, _ := obj.(*unstructured.Unstructured)
		if !r.Assert().NotNilf(unstructuredObj, "YAML document #%d in file %s is not a Kubernetes object", docCounter, fileName) {
			continue
		}

		r.Assert().NotNilf(resources.LookupResource(unstructuredObj.GroupVersionKind()), "YAML document #%d in file %s defines object of kind %s not known in schema", docCounter, fileName, unstructuredObj.GroupVersionKind())
		objs = append(objs, *unstructuredObj)
	}

	// The only acceptable error is EOF.
	if !errors.Is(err, io.EOF) {
		r.Assert().NoErrorf(err, "reading multi-document YAML file %s", fileName)
	}

	// Validate that there is at most a single empty document, and only if the file is otherwise empty.
	if len(objs) > 0 {
		// We can tolerate an empty document at the beginning and at the end
		if len(emptyDocs) > 0 && emptyDocs[0] == 1 {
			emptyDocs = emptyDocs[1:]
		}
		if len(emptyDocs) > 0 && emptyDocs[len(emptyDocs)-1] == docCounter {
			emptyDocs = emptyDocs[:len(emptyDocs)-1]
		}
		r.Assert().Empty(emptyDocs, "multi-document YAML file %s is non-empty but has empty documents", fileName)
	} else {
		r.Assert().LessOrEqualf(len(emptyDocs), 1, "multi-document YAML file %s has multiple empty documents", fileName)
	}

	return objs
}

func (r *runner) instantiateWorld(renderVals chartutil.Values, resources openapi.Resources) map[string]interface{} {
	world := make(map[string]interface{})

	renderValsBytes, err := json.Marshal(renderVals)
	if err != nil {
		panic(errors.Wrap(err, "marshaling Helm render values to JSON"))
	}
	var helmRenderVals map[string]interface{}
	if err := json.Unmarshal(renderValsBytes, &helmRenderVals); err != nil {
		panic(errors.Wrap(err, "unmarshaling Helm render values"))
	}
	world["helm"] = helmRenderVals

	renderedTemplates, err := (&engine.Engine{}).Render(r.tgt.Chart, renderVals)

	if *r.test.ExpectError {
		r.Require().Error(err, "expected rendering to fail")
		errStr := err.Error()
		// Store the error string normalized, to avoid impact of formatting.
		world["error"] = normalizeString(errStr)
		world["errorRaw"] = errStr
		return world
	}

	r.Require().NoError(err, "expected rendering to succeed")

	var allObjects []interface{}

	for fileName, renderedContents := range renderedTemplates {
		// TODO: Support subcharts (even though we don't use them)
		if !r.Assert().Truef(stringutils.ConsumePrefix(&fileName, r.tgt.Chart.Name()+"/"), "unexpected file %s", fileName) {
			continue
		}

		if fileName == "templates/NOTES.txt" {
			world["notes"] = normalizeString(renderedContents)
			world["notesRaw"] = renderedContents
			continue
		}

		if !r.Assert().Equalf(".yaml", path.Ext(fileName), "unexpected file type for file %s", fileName) {
			continue
		}

		objs := r.readAndValidateYAML(fileName, renderedContents, resources)

		for _, obj := range objs {
			kindPlural := strings.TrimSuffix(strings.ToLower(obj.GetKind()), "s") + "s"
			objsByKind, _ := world[kindPlural].(map[string]interface{})
			if objsByKind == nil {
				objsByKind = make(map[string]interface{})
				world[kindPlural] = objsByKind
			}
			if !r.Assert().NotContainsf(objsByKind, obj.GetName(), "duplicate object %s/%s", kindPlural, obj.GetName()) {
				continue
			}
			objsByKind[obj.GetName()] = obj.Object
			allObjects = append(allObjects, obj.Object)
		}
	}

	world["objects"] = allObjects
	return world
}

func (r *runner) loadSchemas() (visible, available schemas.Schemas) {
	var visibleSchemaNames, availableSchemaNames []string
	r.test.forEachScopeTopDown(func(t *Test) {
		server := t.Server
		if server == nil {
			return
		}
		if server.NoInherit {
			visibleSchemaNames = nil
			availableSchemaNames = nil
		}
		for _, schemaName := range server.AvailableSchemas {
			schemaName = strings.ToLower(schemaName)
			availableSchemaNames = append(availableSchemaNames, schemaName)
		}
		for _, schemaName := range server.VisibleSchemas {
			schemaName = strings.ToLower(schemaName)
			visibleSchemaNames = append(visibleSchemaNames, schemaName)
			// Every visible schema is also available (but not vice versa)
			availableSchemaNames = append(availableSchemaNames, schemaName)
		}
	})

	availableSchemaNames = sliceutils.StringUnique(availableSchemaNames)
	visibleSchemaNames = sliceutils.StringUnique(visibleSchemaNames)

	schemaRegistry := r.tgt.SchemaRegistry
	if schemaRegistry == nil {
		schemaRegistry = schemas.BuiltinSchemas()
	}

	for _, schemaName := range availableSchemaNames {
		schema, err := schemaRegistry.GetSchema(schemaName)
		r.Require().NoErrorf(err, "failed to load schema %q", schemaName)
		available = append(available, schema)
	}
	for _, schemaName := range visibleSchemaNames {
		schema, err := schemaRegistry.GetSchema(schemaName)
		r.Require().NoErrorf(err, "failed to load schema %q", schemaName)
		visible = append(visible, schema)
	}

	return visible, available
}

func (r *runner) Run() {
	var values map[string]interface{}
	releaseOpts := r.tgt.ReleaseOptions

	r.test.forEachScopeBottomUp(func(t *Test) {
		scopeVals := t.Values
		if len(scopeVals) == 0 {
			return
		}
		values = chartutil.CoalesceTables(values, runtime.DeepCopyJSON(scopeVals))
	})
	r.test.forEachScopeTopDown(func(t *Test) {
		rel := t.Release
		if rel == nil {
			return
		}
		rel.apply(&releaseOpts)
	})

	visibleSchemas, availableSchemas := r.loadSchemas()

	caps := r.tgt.Capabilities
	if caps == nil {
		caps = chartutil.DefaultCapabilities
	}
	if len(visibleSchemas) > 0 {
		newCaps := *caps
		newCaps.APIVersions = visibleSchemas.VersionSet()
		caps = &newCaps
	}

	r.test.forEachScopeTopDown(func(t *Test) {
		if t.Capabilities == nil {
			return
		}

		newCaps := *caps
		newCaps.KubeVersion = t.Capabilities.toHelmKubeVersion()
		caps = &newCaps
	})

	renderVals, err := chartutil.ToRenderValues(r.tgt.Chart, values, releaseOpts, caps)
	r.Require().NoError(err, "failed to obtain render values")

	world := r.instantiateWorld(renderVals, availableSchemas)
	r.evaluatePredicates(world)
}

func (r *runner) evaluatePredicates(world map[string]interface{}) {
	var allFuncDefs []*gojq.FuncDef
	var allPreds []*gojq.Query

	r.test.forEachScopeTopDown(func(t *Test) {
		allFuncDefs = append(allFuncDefs, t.funcDefs...)
		for _, pred := range t.predicates {
			predWithFuncs := *pred
			predWithFuncs.FuncDefs = make([]*gojq.FuncDef, 0, len(allFuncDefs)+len(pred.FuncDefs))
			predWithFuncs.FuncDefs = append(predWithFuncs.FuncDefs, allFuncDefs...)
			predWithFuncs.FuncDefs = append(predWithFuncs.FuncDefs, pred.FuncDefs...)
			allPreds = append(allPreds, &predWithFuncs)
		}
	})

	for _, pred := range allPreds {
		code, err := compiler.Compile(pred, gojq.WithVariables([]string{"$_"}))

		if !r.Assert().NoErrorf(err, "failed to compile predicate %q", pred) {
			continue
		}

		worldCopy := runtime.DeepCopyJSON(world)
		iter := code.Run(worldCopy, worldCopy)
		hadElem := false
		for result, ok := iter.Next(); ok; result, ok = iter.Next() {
			hadElem = true
			err, _ := result.(error)
			if errors.Is(err, logic.ErrAssumptionViolation) {
				continue
			}
			if !r.Assert().NoErrorf(err, "failed to evaluate pred %q", pred) {
				continue
			}
			r.Assert().True(logic.Truthy(result), "predicate %q evaluated to falsy result %v", pred, result)
		}
		r.Assert().Truef(hadElem, "predicate %q evaluated to empty sequence", pred)
	}
}
