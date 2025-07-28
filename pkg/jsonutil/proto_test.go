package jsonutil

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/compat.alert.json
var compactJson string

//go:embed testdata/pretty.alert.json
var prettyJson string

func TestNil(t *testing.T) {
	output, err := MarshalToString(nil)
	assert.Equal(t, errNil, err)
	assert.Empty(t, output)

	buff := &bytes.Buffer{}
	err = MarshalPretty(buff, nil)
	assert.Equal(t, errNil, err)
	assert.Empty(t, buff)

	buff = &bytes.Buffer{}
	err = Marshal(buff, nil)
	assert.Equal(t, errNil, err)
	assert.Empty(t, buff)
}

func TestMarshal(t *testing.T) {
	output := &bytes.Buffer{}
	err := Marshal(output, alert())
	assert.NoError(t, err)
	assert.Equal(t, compactJson, output.String())
}

func TestMarshalPretty(t *testing.T) {
	output := &bytes.Buffer{}
	err := MarshalPretty(output, alert())
	assert.NoError(t, err)
	assert.Equal(t, prettyJson, output.String())
}

func TestMarshalString(t *testing.T) {
	output, err := MarshalToString(alert())
	assert.NoError(t, err)
	assert.Equal(t, prettyJson, output)
}

func alert() *storage.Alert {
	input := fixtures.GetAlert()
	input.Time = protocompat.GetProtoTimestampZero()
	return input
}
