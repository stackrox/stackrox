package uuid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	id = `b455a167-2302-4d37-b41e-f1b4092da5e9`
)

func TestValidity(t *testing.T) {
	cases := []struct {
		name        string
		id          string
		errExpected bool
	}{
		{
			name:        "empty",
			id:          "",
			errExpected: true,
		},
		{
			name:        "generatedV4",
			id:          NewV4().String(),
			errExpected: false,
		},
		{
			name:        "Almost valid ID",
			id:          "b455a167-2302-4d37-b41e-f1b4092da5e",
			errExpected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := FromString(c.id)
			assert.Equal(t, c.errExpected, err != nil, "Got error %s", err)
		})
	}
}

func TestFromString(t *testing.T) {
	first, err := FromString(id)
	require.Nil(t, err)
	second, err := FromString(id)
	require.Nil(t, err)

	if first != second {
		t.Errorf("Identical UUID were not equal; %s; %s", first, second)
	}

	idMap := make(map[UUID]bool)

	idMap[first] = true

	if _, found := idMap[second]; !found {
		t.Errorf("Couldn't find UUID, %s, in map", second)
	}
}

func TestNewV7(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		id := NewV7()
		_, err := FromString(id.String())
		assert.NoError(t, err)
	})

	t.Run("version nibble is 7", func(t *testing.T) {
		id := NewV7()
		// Version is the 13th hex character (index 14 in the string, after the second hyphen).
		assert.Equal(t, byte('7'), id.String()[14])
	})

	t.Run("unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for range 100 {
			id := NewV7().String()
			assert.False(t, seen[id], "duplicate UUID generated: %s", id)
			seen[id] = true
		}
	})

	t.Run("monotonically increasing", func(t *testing.T) {
		prev := NewV7().String()
		for range 100 {
			curr := NewV7().String()
			assert.Greater(t, curr, prev, "UUIDv7 should be monotonically increasing")
			prev = curr
		}
	})
}

func TestNewTestUUID(t *testing.T) {
	test := NewTestUUID(-1)
	require.NotNil(t, test)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", test.String())

	test = NewTestUUID(1)
	require.NotNil(t, test)
	assert.Equal(t, "00010001-0001-0001-0001-000100010001", test.String())

	test = NewTestUUID(10)
	require.NotNil(t, test)
	assert.Equal(t, "00100010-0010-0010-0010-001000100010", test.String())

	test = NewTestUUID(100)
	require.NotNil(t, test)
	assert.Equal(t, "01000100-0100-0100-0100-010001000100", test.String())

	test = NewTestUUID(1000)
	require.NotNil(t, test)
	assert.Equal(t, "10001000-1000-1000-1000-100010001000", test.String())

	test = NewTestUUID(1111)
	require.NotNil(t, test)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", test.String())

	test = NewTestUUID(10000)
	require.NotNil(t, test)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", test.String())
}
