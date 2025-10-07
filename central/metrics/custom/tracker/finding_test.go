package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testFindingTypeBase struct {
	value string
}

func TestNewFindingCollector_base(t *testing.T) {
	n := 0
	yield := func(f *testFindingTypeBase, _ error) bool {
		n++
		return f.value != "stop value"
	}
	var f testFindingTypeBase
	collector := NewFindingCollector(yield)

	t.Run("initial values", func(t *testing.T) {
		assert.Zero(t, n)
		require.NotNil(t, f)
		assert.Empty(t, f.value)
	})

	t.Run("no error", func(t *testing.T) {
		f.value = "test value"
		err := collector.Yield(&f)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
	})

	t.Run("yield stops", func(t *testing.T) {
		f.value = "stop value"
		err := collector.Yield(&f)
		assert.ErrorIs(t, err, ErrStopIterator)
		assert.Equal(t, 2, n)
	})
}

func TestCollector_errors(t *testing.T) {
	var finding *testFindingTypeBase
	var err error
	var collector Collector[*testFindingTypeBase] = func(f *testFindingTypeBase, e error) bool {
		finding, err = f, e
		return true
	}

	t.Run("random error", func(t *testing.T) {
		collector.Error(errInvalidConfiguration)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Nil(t, finding)
	})

	t.Run("nil error", func(t *testing.T) {
		err = errInvalidConfiguration
		collector.Error(nil)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Nil(t, finding)
	})

	t.Run("stop iterator", func(t *testing.T) {
		collector.Error(ErrStopIterator)
		assert.ErrorIs(t, err, ErrStopIterator)
		assert.Nil(t, finding)
	})

	t.Run("finally", func(t *testing.T) {
		collector.Finally(errInvalidConfiguration)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Nil(t, finding)
	})

	t.Run("finally", func(t *testing.T) {
		err = nil
		collector.Finally(ErrStopIterator)
		assert.NoError(t, err)
		assert.Nil(t, finding)
	})
}
