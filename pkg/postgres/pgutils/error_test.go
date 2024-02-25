package pgutils

import (
	"context"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestWrappedErrors(t *testing.T) {
	cases := []struct {
		err       error
		transient bool
	}{
		{
			err:       errors.New("hello"),
			transient: false,
		},
		{
			err:       errors.Wrap(errors.New("hello"), "hello"),
			transient: false,
		},
		{
			err:       errors.Wrap(context.DeadlineExceeded, "hello"),
			transient: true,
		},
		{
			err:       errors.Wrap(errors.Wrap(io.EOF, "1"), "2"),
			transient: true,
		},
		{
			err:       errors.Wrap(errors.Wrap(errors.New("nothing"), "1"), "2"),
			transient: false,
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.transient, IsTransientError(c.err))
	}
}
