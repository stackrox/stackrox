package tracker

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testFindingTypeBase struct {
	FindingBase
	value string
}

type testFindingTypeErr struct {
	FindingWithErr
	value string
}

func TestNewFindingCollector_base(t *testing.T) {
	n := 0
	yield := func(f *testFindingTypeBase) bool {
		n++
		return f.value != "error value"
	}
	var f testFindingTypeBase
	collector := NewFindingCollector(yield)

	t.Run("initial values", func(t *testing.T) {
		assert.Zero(t, n)
		require.NotNil(t, f)
		assert.NoError(t, f.GetError())
		assert.Equal(t, 1, f.GetIncrement())
		assert.Empty(t, f.value)
	})

	t.Run("no error", func(t *testing.T) {
		f.value = "test value"
		err := collector(&f)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
	})

	t.Run("yield stops", func(t *testing.T) {
		f.value = "error value"
		err := collector(&f)
		assert.ErrorIs(t, err, ErrStopIterator)
		assert.Equal(t, 2, n)
	})
}

func TestNewFindingCollector_withErr(t *testing.T) {
	n := 0
	yield := func(f *testFindingTypeErr) bool {
		n++
		return f.value != "error value"
	}

	var f testFindingTypeErr
	collector := NewFindingCollector(yield)

	t.Run("initial values", func(t *testing.T) {
		assert.Zero(t, n)
		require.NotNil(t, f)
		assert.NoError(t, f.GetError())
		assert.Equal(t, 1, f.GetIncrement())
		assert.Empty(t, f.value)
	})

	t.Run("no error", func(t *testing.T) {
		f.value = "test value"
		err := collector(&f)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
	})

	t.Run("yield stops", func(t *testing.T) {
		f.value = "error value"
		err := collector(&f)
		assert.ErrorIs(t, err, ErrStopIterator)
		assert.Equal(t, 2, n)
	})

	t.Run("finally do not yield if no error", func(t *testing.T) {
		f.SetError(nil)
		collector.Finally(&f)
		assert.Equal(t, 2, n)
	})

	t.Run("finally do not yield if ErrStopIterator", func(t *testing.T) {
		f.SetError(ErrStopIterator)
		collector.Finally(&f)
		assert.Equal(t, 2, n)
	})

	t.Run("finally yield error", func(t *testing.T) {
		f.SetError(errors.New("some error"))
		collector.Finally(&f)
		assert.Equal(t, 3, n)
	})
}
