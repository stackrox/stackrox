package errorhelpers

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAs(t *testing.T) {
	el := NewErrorList("start")
	err := errors.WithMessage(errox.MakeSensitive("******", errors.New("SECRET")), "wrapped")
	el.AddError(err)
	el.AddError(errox.AlreadyExists)

	var rerr *errox.RoxError

	require.True(t, errors.As(el, &rerr))
	assert.Equal(t, "already exists", rerr.Error())

	var serr errox.SensitiveError

	require.True(t, errors.As(el, &serr))
	assert.Equal(t, "start errors: [wrapped: ******, already exists]", el.Error())
	assert.Equal(t, "start errors: [wrapped: SECRET, already exists]", errox.GetSensitiveError(el))
}
