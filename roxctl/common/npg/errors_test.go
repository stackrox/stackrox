package npg

import (
	goerrors "errors"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type standardNPGuardError struct {
	err      error
	location string
	isSevere bool
}

func (e *standardNPGuardError) IsFatal() bool {
	return false
}

func (e *standardNPGuardError) Error() error {
	return e.err
}

func (e *standardNPGuardError) Location() string {
	return e.location
}

func (e *standardNPGuardError) IsSevere() bool {
	return e.isSevere
}

func warning(txt, loc string) *standardNPGuardError {
	return &standardNPGuardError{
		err:      errors.New(txt),
		location: loc,
		isSevere: false,
	}
}
func err(txt, loc string) *standardNPGuardError {
	return &standardNPGuardError{
		err:      errors.New(txt),
		location: loc,
		isSevere: true,
	}
}

func TestHandleNPGerrors(t *testing.T) {
	tests := map[string]struct {
		errors    []*standardNPGuardError
		wantWarns []string
		wantErrs  []string
	}{
		"warnings have location correctly attached": {
			errors: []*standardNPGuardError{
				warning("foo", "/root"),
				warning("bar", "/root"),
			},
			wantWarns: []string{"foo (at \"/root\")", "bar (at \"/root\")"},
		},
		"errors have location correctly attached": {
			errors: []*standardNPGuardError{
				err("foo", "/root"),
			},
			wantErrs:  []string{"foo (at \"/root\")"},
			wantWarns: []string{},
		},
		"warnings and errors are correctly classified": {
			errors: []*standardNPGuardError{
				err("foo", "/root"),
				warning("bar", "/root"),
			},
			wantErrs:  []string{"foo (at \"/root\")"},
			wantWarns: []string{"bar (at \"/root\")"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotWarns, gotErrs := HandleNPGuardErrors(tt.errors)
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

func TestSummarizeErrors(t *testing.T) {
	logio, _, _, _ := io.TestIO()
	l := logger.NewLogger(logio, printer.DefaultColorPrinter())

	tests := map[string]struct {
		warns                 []error
		errs                  []error
		treatWarningsAsErrors bool
		wantErr               error
	}{
		"error marker is added when errors are present": {
			warns:                 nil,
			errs:                  []error{errors.New("foo")},
			treatWarningsAsErrors: false,
			wantErr:               ErrErrors,
		},
		"warning marker in the errors array should be treated as an error": {
			warns:                 nil,
			errs:                  []error{ErrWarnings},
			treatWarningsAsErrors: false,
			wantErr:               ErrErrors,
		},
		"warning marker is added when warnings are present and treatWarningsAsErrors is true": {
			warns:                 []error{errors.New("foo")},
			errs:                  nil,
			treatWarningsAsErrors: true,
			wantErr:               ErrWarnings,
		},
		"no marker is added when warnings are present and treatWarningsAsErrors is false": {
			warns:                 []error{errors.New("foo")},
			errs:                  nil,
			treatWarningsAsErrors: false,
			wantErr:               nil,
		},
		"no marker is added for no warnings and no errors": {
			warns:                 nil,
			errs:                  nil,
			treatWarningsAsErrors: false,
			wantErr:               nil,
		},
		"error marker has precedence over the warning marker": {
			warns:                 []error{errors.New("foo")},
			errs:                  []error{errors.New("bar")},
			treatWarningsAsErrors: true,
			wantErr:               ErrErrors,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotErr := SummarizeErrors(tt.warns, tt.errs, tt.treatWarningsAsErrors, l)
			assert.ErrorIs(t, gotErr, tt.wantErr)
		})
	}
}
