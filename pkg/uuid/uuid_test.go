package uuid

import (
	"strings"
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

func FuzzFromString(f *testing.F) {
	// Seed with valid UUID formats according to UnmarshalText documentation
	f.Add("6ba7b810-9dad-11d1-80b4-00c04fd430c8")            // canonical format
	f.Add("{6ba7b810-9dad-11d1-80b4-00c04fd430c8}")          // with braces
	f.Add("urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8")   // URN format
	f.Add("b455a167-2302-4d37-b41e-f1b4092da5e9")            // from test constant
	f.Add("")                                                // empty string
	f.Add("invalid")                                         // invalid format
	f.Add("00000000-0000-0000-0000-000000000000")            // nil UUID
	f.Add("FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF")            // max UUID
	f.Add("6ba7b810-9dad-11d1-80b4-00c04fd430c8-extra")      // extra characters
	f.Add("6ba7b810_9dad_11d1_80b4_00c04fd430c8")            // wrong separator
	f.Add("{6ba7b810-9dad-11d1-80b4-00c04fd430c8")           // unbalanced brace
	f.Add("6ba7b810-9dad-11d1-80b4-00c04fd430c8}")           // unbalanced brace
	f.Add("urn:uuid:{6ba7b810-9dad-11d1-80b4-00c04fd430c8}") // mixed formats
	f.Add("6ba7b810-9dad-11d1-80b4-00c04fd430c")             // too short
	f.Add("6ba7b810-9dad-11d1-80b4-00c04fd430c88")           // too long
	f.Add(NewV4().String())                                  // random valid UUID
	f.Add(NewDummy().String())                               // dummy test UUID
	f.Add(strings.Repeat("a", 1000))                         // very long string

	f.Fuzz(func(_ *testing.T, input string) {
		u, err := FromString(input)
		if err == nil {
			// Verify round-trip consistency
			_, _ = FromString(u.String())
			_ = u.Bytes()
		}
	})
}

func FuzzUnmarshalBinary(f *testing.F) {
	// Seed with valid binary UUIDs
	validUUID, _ := FromString("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	f.Add(validUUID.Bytes()) // valid 16-byte UUID
	f.Add(Nil.Bytes())       // nil UUID
	f.Add(NewV4().Bytes())   // random UUID
	f.Add([]byte{})          // empty
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) // all bits set
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})         // sequential
	f.Add([]byte{1, 2, 3})                                                      // too short
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}) // too long

	f.Fuzz(func(_ *testing.T, data []byte) {
		var u UUID
		err := u.UnmarshalBinary(data)
		if err == nil {
			_, _ = u.MarshalBinary()
			_ = u.Bytes()
		}
	})
}

func FuzzUnmarshalText(f *testing.F) {
	// Seed with valid text formats
	f.Add([]byte("6ba7b810-9dad-11d1-80b4-00c04fd430c8"))
	f.Add([]byte("{6ba7b810-9dad-11d1-80b4-00c04fd430c8}"))
	f.Add([]byte("urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"))
	f.Add([]byte(""))
	f.Add([]byte("invalid"))
	f.Add([]byte("00000000-0000-0000-0000-000000000000"))

	f.Fuzz(func(_ *testing.T, text []byte) {
		var u UUID
		err := u.UnmarshalText(text)
		if err == nil {
			marshaled, _ := u.MarshalText()
			var u2 UUID
			_ = u2.UnmarshalText(marshaled)
		}
	})
}
