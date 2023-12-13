package authn

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestExtractorErrorNilGuards(t *testing.T) {
	var err *ExtractorError

	assert.Equal(t, "", err.Error())
	assert.Nil(t, err.Unwrap())
	assert.NotPanics(t, func() {
		err.LogL(requestinfo.RequestInfo{})
	})
}

func TestExtractorErrorUnwrap(t *testing.T) {
	jwtErr := jwt.ErrExpired
	err := NewExtractorError("test", "test msg", jwtErr)
	assert.ErrorIs(t, err.Unwrap(), jwt.ErrExpired)
}

func TestExtractorErrorRootErrorNotExposed(t *testing.T) {
	rootErrMsg := "root"
	errMsg := "error-msg"

	internalErr := errors.New(rootErrMsg)
	err := NewExtractorError("test", errMsg, internalErr)

	assert.NotContains(t, err.Error(), rootErrMsg)
	assert.Contains(t, err.Error(), errMsg)

	errNoCreds := errox.NoCredentials.CausedBy(err)
	assert.NotContains(t, errNoCreds.Error(), rootErrMsg)
	assert.Contains(t, errNoCreds.Error(), errMsg)
}

func TestExtractorErrorRootErrorNotExposedByErrox(t *testing.T) {
	rootErrMsg := "root"
	errMsg := "error-msg"

	internalErr := errors.New(rootErrMsg)
	err := NewExtractorError("test", errMsg, internalErr)
	errNoCreds := errox.NoCredentials.CausedBy(err)

	assert.NotContains(t, errNoCreds.Error(), rootErrMsg)
	assert.Contains(t, errNoCreds.Error(), errMsg)
}

func TestExtractorErrorRootErrorNil(t *testing.T) {
	errMsg := "error-msg"
	err := NewExtractorError("test", errMsg, nil)

	assert.Nil(t, err.Unwrap())
	assert.Contains(t, err.Error(), errMsg)
	assert.NotPanics(t, func() {
		err.LogL(requestinfo.RequestInfo{Hostname: "example.com"})
	})
}
