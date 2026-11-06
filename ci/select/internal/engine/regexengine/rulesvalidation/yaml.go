package rulesvalidation

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
	"go.yaml.in/yaml/v3"
)

// YAML checks only that a document parses as a single YAML document.
// It proves nothing about rule semantics, so the engine still rejects
// whatever it cannot compile.
type YAML struct{}

var _ engine.Validator = YAML{}

func (YAML) Validate(data []byte) ([]string, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, errors.Wrap(err, "decode rules")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("rules must contain a single YAML document")
	}
	return nil, nil
}
