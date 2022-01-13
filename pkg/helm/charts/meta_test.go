package charts

import (
	"testing"
	"text/template"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMetaValuesStringKeyCompatibility(t *testing.T) {
	m := MetaValues{"foo": "bar"}
	assert.Equal(t, "bar", m["foo"])

	m["foo"] = "No" + "vem" + "ber"
	assert.Equal(t, "November", m["foo"])

	str := "blah"
	// Can't use string key directly, need to cast it, otherwise the code does not compile.
	m[MetaValuesKey(str)] = 6

	assert.Equal(t, 6, m["blah"])             // String constants are casted automatically.
	assert.Equal(t, 6, m[MetaValuesKey(str)]) // String variables need to be casted explicitly.
}

func TestMetaValuesToRaw(t *testing.T) {
	m := MetaValues{"foo": "bar", "baz": 6}

	raw := m.ToRaw()

	assert.Equal(t, map[string]interface{}{"baz": 6, "foo": "bar"}, raw)

	assert.NotSame(t, raw, m)

	// This might seem a bit surprising, but assert is using reflect.DeepEqual which checks if types are matching.
	assert.NotEqual(t, raw, m)
}

func TestMetaValuesInTemplating(t *testing.T) {
	tpl := template.Must(template.New("blah").Parse("value is: {{.foo}}"))

	// This fragment shows that we can use map[string]interface{} (hand-crafted) in templates without issues.
	dataMap := map[string]interface{}{"foo": 6}
	res, err := templates.ExecuteToString(tpl, dataMap)
	assert.NoError(t, err)
	assert.Equal(t, "value is: 6", res)

	// This fragment shows that an attempt to use strongly-typed MetaValues leads to a rendering error.
	dataMetaVals := MetaValues{"foo": 500}
	res, err = templates.ExecuteToString(tpl, dataMetaVals)
	assert.Equal(t, "", res)
	assert.Error(t, err)
	assert.Equal(t, "template: blah:1:12: executing \"blah\" at <.foo>: can't evaluate field foo in type charts.MetaValues", err.Error())

	// This fragment shows what to do with MetaValues in order to leverage them in templating.
	dataConvertedToRaw := dataMetaVals.ToRaw()
	res, err = templates.ExecuteToString(tpl, dataConvertedToRaw)
	assert.NoError(t, err)
	assert.Equal(t, "value is: 500", res)
}

// TestRequiredMetaValuesArePresent validates that MetaValues attributes that are consumed and required by .htpl files
// are actually present.
func TestRequiredMetaValuesArePresent(t *testing.T) {
	testutils.SetExampleVersion(t)
	restorer := testbuildinfo.SetForTest(t)
	defer func() {
		restorer.Restore()
	}()

	flavor := flavorUtils.MakeImageFlavorForTest(t)
	cases := map[string]MetaValues{
		"default": GetMetaValuesForFlavor(flavor),
		"rhacs":   RHACSMetaValues(),
	}
	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			assert.NotEmpty(t, c["MainRegistry"])
			assert.NotEmpty(t, c["ImageRemote"])
			assert.NotEmpty(t, c["CollectorRegistry"])
			assert.NotEmpty(t, c["CollectorFullImageRemote"])
			assert.NotEmpty(t, c["CollectorSlimImageRemote"])
			assert.NotEmpty(t, c["CollectorFullImageTag"])
			assert.NotEmpty(t, c["CollectorSlimImageTag"])
			assert.NotEmpty(t, (c["ChartRepo"].(defaults.ChartRepo)).URL)
			assert.NotNil(t, c["ImagePullSecrets"])

			versions := c["Versions"].(version.Versions)
			assert.NotEmpty(t, versions.ChartVersion)
			assert.NotEmpty(t, versions.MainVersion)
			// TODO: replace this with the check of the scanner tag once we migrate to it instead of version.
			assert.NotEmpty(t, versions.ScannerVersion)
		})
	}
}
