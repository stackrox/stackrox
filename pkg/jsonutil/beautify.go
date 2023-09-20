package jsonutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func toJSONAndBeautify(v any) (string, error) {
	jStr, err := json.Marshal(v)
	if err != nil {

		return "", err
	}
	var prettyJSON bytes.Buffer
	if err = json.Indent(&prettyJSON, jStr, "", "\t"); err != nil {
		return "", err
	}
	return prettyJSON.String(), nil
}

// LogAndBeautify converts an object to json and log the beautified json object
func LogAndBeautify(t *testing.T, v any, heading string) {
	str, err := toJSONAndBeautify(v)
	assert.NoError(t, err)
	t.Log(heading)
	t.Log(str)
}
