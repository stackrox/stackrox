package signatureintegration

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

// mockGetter implements Getter for testing.
type mockGetter struct {
	integrations map[string]*storage.SignatureIntegration
	err          error
}

func (m *mockGetter) GetSignatureIntegration(_ context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	if m.err != nil {
		return nil, false, m.err
	}
	integration, found := m.integrations[id]
	return integration, found, nil
}

func TestGetVerifierName(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty string for empty verifier ID", func(t *testing.T) {
		mock := &mockGetter{}

		name, err := GetVerifierName(ctx, mock, "")
		assert.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("returns integration name when found", func(t *testing.T) {
		mock := &mockGetter{
			integrations: map[string]*storage.SignatureIntegration{
				"test-id": {Name: "my-integration"},
			},
		}

		name, err := GetVerifierName(ctx, mock, "test-id")
		assert.NoError(t, err)
		assert.Equal(t, "my-integration", name)
	})

	t.Run("returns empty string when not found", func(t *testing.T) {
		mock := &mockGetter{
			integrations: map[string]*storage.SignatureIntegration{},
		}

		name, err := GetVerifierName(ctx, mock, "unknown-id")
		assert.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("returns error when getter fails", func(t *testing.T) {
		mock := &mockGetter{
			err: errors.New("datastore error"),
		}

		name, err := GetVerifierName(ctx, mock, "test-id")
		assert.Error(t, err)
		assert.Empty(t, name)
	})
}

func TestEnrichVerificationResults(t *testing.T) {
	ctx := context.Background()

	t.Run("handles empty results slice", func(t *testing.T) {
		mock := &mockGetter{}

		var results []*storage.ImageSignatureVerificationResult
		EnrichVerificationResults(ctx, mock, results)
		// No panic, no error
	})

	t.Run("enriches multiple results", func(t *testing.T) {
		mock := &mockGetter{
			integrations: map[string]*storage.SignatureIntegration{
				"id-1": {Name: "integration-1"},
				"id-2": {Name: "integration-2"},
			},
		}

		results := []*storage.ImageSignatureVerificationResult{
			{VerifierId: "id-1"},
			{VerifierId: "id-2"},
		}

		EnrichVerificationResults(ctx, mock, results)

		assert.Equal(t, "integration-1", results[0].GetVerifierName())
		assert.Equal(t, "integration-2", results[1].GetVerifierName())
	})

	t.Run("leaves VerifierName empty when integration not found", func(t *testing.T) {
		mock := &mockGetter{
			integrations: map[string]*storage.SignatureIntegration{},
		}

		results := []*storage.ImageSignatureVerificationResult{
			{VerifierId: "unknown-id"},
		}

		EnrichVerificationResults(ctx, mock, results)

		assert.Empty(t, results[0].GetVerifierName())
	})

	t.Run("continues enriching after error", func(t *testing.T) {
		callCount := 0
		mock := &mockGetter{
			integrations: map[string]*storage.SignatureIntegration{
				"id-2": {Name: "integration-2"},
			},
		}
		// Override to return error for first call only
		errorOnFirstCall := &errorOnFirstCallGetter{
			delegate:  mock,
			callCount: &callCount,
		}

		results := []*storage.ImageSignatureVerificationResult{
			{VerifierId: "id-1"}, // Will error
			{VerifierId: "id-2"}, // Should still be enriched
		}

		EnrichVerificationResults(ctx, errorOnFirstCall, results)

		assert.Empty(t, results[0].GetVerifierName())                  // Error case
		assert.Equal(t, "integration-2", results[1].GetVerifierName()) // Continued after error
	})
}

// errorOnFirstCallGetter returns an error on the first call, then delegates.
type errorOnFirstCallGetter struct {
	delegate  Getter
	callCount *int
}

func (e *errorOnFirstCallGetter) GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	*e.callCount++
	if *e.callCount == 1 {
		return nil, false, errors.New("first call error")
	}
	return e.delegate.GetSignatureIntegration(ctx, id)
}
