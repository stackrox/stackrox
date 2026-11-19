package authn

import (
	"context"
	"testing"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeExtractor is a test stand-in for IdentityExtractor with canned results.
type fakeExtractor struct {
	identity Identity
	err      *ExtractorError
}

func (e *fakeExtractor) IdentityForRequest(_ context.Context, _ requestinfo.RequestInfo) (Identity, *ExtractorError) {
	return e.identity, e.err
}

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
	assert.Equal(t, err.Unwrap(), jwt.ErrExpired)
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

func TestCombineExtractors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	testIdentity := mocks.NewMockIdentity(mockCtrl)
	testExtractorError := NewExtractorError("test", "test msg", nil)

	t.Run("single extractor is returned as-is", func(it *testing.T) {
		single := &fakeExtractor{}
		assert.Same(it, single, CombineExtractors(single))
	})

	for name, tc := range map[string]struct {
		extractors       []IdentityExtractor
		expectedIdentity Identity
		expectedErr      *ExtractorError
	}{
		"identity from the first extractor wins": {
			extractors: []IdentityExtractor{
				&fakeExtractor{identity: testIdentity},
				&fakeExtractor{},
			},
			expectedIdentity: testIdentity,
		},
		"error from the first extractor short-circuits": {
			extractors: []IdentityExtractor{
				&fakeExtractor{err: testExtractorError},
				&fakeExtractor{identity: testIdentity},
			},
			expectedErr: testExtractorError,
		},
		"extractor without identity or error falls through to the next": {
			extractors: []IdentityExtractor{
				&fakeExtractor{},
				&fakeExtractor{identity: testIdentity},
			},
			expectedIdentity: testIdentity,
		},
		"all extractors yield neither identity nor error": {
			extractors: []IdentityExtractor{
				&fakeExtractor{},
				&fakeExtractor{},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			combined := CombineExtractors(tc.extractors...)

			id, err := combined.IdentityForRequest(it.Context(), requestinfo.RequestInfo{})
			assert.Equal(it, tc.expectedErr, err)
			if tc.expectedIdentity == nil {
				assert.Nil(it, id)
			} else {
				assert.Same(it, tc.expectedIdentity, id)
			}
		})
	}

	t.Run("no extractors yields neither identity nor error", func(it *testing.T) {
		combined := CombineExtractors()

		id, err := combined.IdentityForRequest(it.Context(), requestinfo.RequestInfo{})
		assert.Nil(it, err)
		assert.Nil(it, id)
	})

	// Sanity check to ensure the test identity mock is set up as expected.
	require.NotNil(t, testIdentity)
}
