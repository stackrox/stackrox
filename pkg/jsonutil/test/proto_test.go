package test

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/compat.alert.json
var compactJson string

//go:embed testdata/pretty.alert.json
var prettyJson string

func TestNil(t *testing.T) {
	output, err := jsonutil.MarshalToString(nil)
	assert.EqualError(t, err, "Marshal called with nil")
	assert.Empty(t, output)

	buff := &bytes.Buffer{}
	err = jsonutil.MarshalPretty(buff, nil)
	assert.EqualError(t, err, "Marshal called with nil")
	assert.Empty(t, buff)

	buff = &bytes.Buffer{}
	err = jsonutil.Marshal(buff, nil)
	assert.EqualError(t, err, "Marshal called with nil")
	assert.Empty(t, buff)
}

func TestMarshal(t *testing.T) {
	output := &bytes.Buffer{}
	err := jsonutil.Marshal(output, alert())
	assert.NoError(t, err)
	assert.Equal(t, compactJson, output.String())
}

func TestMarshalPretty(t *testing.T) {
	output := &bytes.Buffer{}
	err := jsonutil.MarshalPretty(output, alert())
	assert.NoError(t, err)
	assert.Equal(t, prettyJson, output.String())
}

func TestMarshalString(t *testing.T) {
	output, err := jsonutil.MarshalToString(alert())
	assert.NoError(t, err)
	assert.Equal(t, prettyJson, output)
}

func alert() *storage.Alert {
	input := fixtures.GetAlert()
	input.SetTime(protocompat.GetProtoTimestampZero())
	return input
}
