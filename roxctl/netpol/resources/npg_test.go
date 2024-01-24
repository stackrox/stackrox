package resources

import (
	goerrors "errors"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type error1 struct {
	err      error
	location string
	isSevere bool
}

func (e *error1) Error() error {
	return e.err
}

func (e *error1) Location() string {
	return e.location
}

func (e *error1) IsSevere() bool {
	return e.isSevere
}

func warning(txt, loc string) *error1 {
	return &error1{
		err:      errors.New(txt),
		location: loc,
		isSevere: false,
	}
}
func err(txt, loc string) *error1 {
	return &error1{
		err:      errors.New(txt),
		location: loc,
		isSevere: true,
	}
}

func TestHandleNPGerrors(t *testing.T) {
	tests := map[string]struct {
		errors                []ErrorLocationSeverity
		treatWarningsAsErrors bool
		wantWarns             []string
		wantErrs              []string
	}{
		"errors and warnings are correctly classified": {
			errors: []ErrorLocationSeverity{
				warning("foo", "/root"),
				warning("bar", "/root"),
			},
			treatWarningsAsErrors: false,
			wantWarns:             []string{"foo", "bar"},
		},
		"errors should get a marker error added": {
			errors: []ErrorLocationSeverity{
				err("foo", "/root"),
				warning("bar", "/root"),
			},
			treatWarningsAsErrors: false,
			wantErrs:              []string{"foo", "there were errors"},
			wantWarns:             []string{"bar"},
		},
		"warnings should get a marker error added when run with treatWarningsAsErrors": {
			errors: []ErrorLocationSeverity{
				warning("foo", "/root"),
			},
			treatWarningsAsErrors: true,
			wantErrs:              []string{"there were warnings"},
			wantWarns:             []string{"foo"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotWarns, gotErrs := HandleNPGerrors(tt.errors, tt.treatWarningsAsErrors)
			require.Lenf(t, gotWarns, len(tt.wantWarns), "got: %v", goerrors.Join(gotWarns...))
			require.Lenf(t, gotErrs, len(tt.wantErrs), "got: %v", goerrors.Join(gotErrs...))

			for i, ww := range tt.wantWarns {
				assert.ErrorContains(t, gotWarns[i], ww)
			}
			for i, we := range tt.wantErrs {
				assert.ErrorContains(t, gotErrs[i], we)
			}
		})
	}
}
