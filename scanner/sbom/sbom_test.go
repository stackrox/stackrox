package sbom

import (
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/test"
	"github.com/stretchr/testify/assert"
)

func TestGetSBOM(t *testing.T) {
	t.Run("error on nil index report", func(t *testing.T) {
		ctx := test.Logging(t)
		s := NewSBOMer()
		_, err := s.GetSBOM(ctx, nil, nil)

		assert.ErrorContains(t, err, "index report is required")
	})

	t.Run("error on nil opts", func(t *testing.T) {
		ctx := test.Logging(t)
		ir := &claircore.IndexReport{}
		s := NewSBOMer()
		_, err := s.GetSBOM(ctx, ir, nil)

		assert.ErrorContains(t, err, "opts is required")
	})

	t.Run("success", func(t *testing.T) {
		ctx := test.Logging(t)
		ir := &claircore.IndexReport{}
		s := NewSBOMer()

		sbom, err := s.GetSBOM(ctx, ir, &Options{})
		assert.NoError(t, err)
		assert.NotEmpty(t, sbom)
	})
}
