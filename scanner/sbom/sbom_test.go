package sbom

import (
	"context"
	"testing"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/assert"
)

func TestGetSBOM(t *testing.T) {
	t.Run("error on nil index report", func(t *testing.T) {
		s := NewSBOMer()
		_, err := s.GetSBOM(context.Background(), nil, nil)

		assert.ErrorContains(t, err, "index report is required")
	})

	t.Run("error on nil opts", func(t *testing.T) {
		ir := &claircore.IndexReport{}
		s := NewSBOMer()
		_, err := s.GetSBOM(context.Background(), ir, nil)

		assert.ErrorContains(t, err, "opts is required")
	})

	t.Run("success", func(t *testing.T) {
		ir := &claircore.IndexReport{}
		s := NewSBOMer()

		sbom, err := s.GetSBOM(context.Background(), ir, &Options{})
		assert.NoError(t, err)
		assert.NotEmpty(t, sbom)
	})
}
