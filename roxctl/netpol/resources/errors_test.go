package resources

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

func TestHandleAggregatedError(t *testing.T) {
	tests := map[string]struct {
		treatWarningsAsErrors bool
		inputErr              utilerrors.Aggregate
		wantWarn              []string
		wantErr               []string
	}{
		"No input should result in no warnings and no errors": {
			treatWarningsAsErrors: false,
			inputErr:              nil,
			wantWarn:              []string{},
			wantErr:               []string{},
		},
		"Nil errors should be ignored": {
			treatWarningsAsErrors: false,
			inputErr: utilerrors.NewAggregate([]error{
				errors.New("error 1"),
				nil,
				errors.New("error 2"),
			}),
			wantWarn: []string{},
			wantErr:  []string{"error 1", "error 2"},
		},
		"Specific errors should be recognized as warnings": {
			treatWarningsAsErrors: false,
			inputErr: utilerrors.NewAggregate([]error{
				errors.New("unable to decode \"/var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml\": json: cannot unmarshal string into Go value of type unstructured.detector"),
				errors.New("error parsing /var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml: error converting YAML to JSON: yaml: line 16: found character that cannot start any token"),
				errors.New("not a warning"),
			}),
			wantWarn: []string{
				"unable to decode \"/var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml\": json: cannot unmarshal string into Go value of type unstructured.detector",
				"error parsing /var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml: error converting YAML to JSON: yaml: line 16: found character that cannot start any token",
			},
			wantErr: []string{"not a warning"},
		},

		"Treating warnings as errors should produce errors with marker error at the end": {
			treatWarningsAsErrors: true,
			inputErr: utilerrors.NewAggregate([]error{
				errors.New("unable to decode \"/var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml\": json: cannot unmarshal string into Go value of type unstructured.detector"),
				errors.New("error parsing /var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml: error converting YAML to JSON: yaml: line 16: found character that cannot start any token"),
				errors.New("not a warning"),
			}),
			wantWarn: []string{
				"unable to decode \"/var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml\": json: cannot unmarshal string into Go value of type unstructured.detector",
				"error parsing /var/folders/gn/_c1mk30n4w71g65nq_zb58w40000gn/T/tmp.eWb132tVAW/templated-01-XXXXXX-file1.yaml: error converting YAML to JSON: yaml: line 16: found character that cannot start any token",
			},
			wantErr: []string{
				// true error first
				"not a warning",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotWarn, gotErr := handleAggregatedError(tt.inputErr)
			require.Lenf(t, gotWarn, len(tt.wantWarn), "expected %d warnings but got %d", len(tt.wantWarn), len(gotWarn))
			require.Lenf(t, gotErr, len(tt.wantErr), "expected %d errors but got %d", len(tt.wantErr), len(gotErr))

			for i := range gotWarn {
				assert.ErrorContains(t, gotWarn[i], tt.wantWarn[i])
			}
			for i := range gotErr {
				assert.ErrorContains(t, gotErr[i], tt.wantErr[i])
			}
		})
	}
}
