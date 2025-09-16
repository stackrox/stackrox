package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMust_NoErrs(t *testing.T) {

	assert.NotPanics(t, func() {
		Must()
	})
}

func TestMust_AllNilErrs(t *testing.T) {

	assert.NotPanics(t, func() {
		Must(nil, nil, nil)
	})
}

func TestMust_OneNonNilErr(t *testing.T) {

	assert.Panics(t, func() {
		Must(nil, errors.New("some error"), nil)
	})
}
