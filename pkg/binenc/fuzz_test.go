package binenc

import (
	"bytes"
	"testing"
)

// FuzzDecodeBytesList feeds arbitrary bytes to DecodeBytesList and ensures it doesn't panic.
// This tests the decoder's robustness against malformed input.
func FuzzDecodeBytesList(f *testing.F) {
	// Seed with valid encoded data from the existing test
	f.Add(EncodeBytesList([]byte("foobar\x00baz"), []byte("\x00\x01\x00")))

	// Seed with empty input
	f.Add([]byte{})

	// Seed with single byte slice
	f.Add(EncodeBytesList([]byte("test")))

	// Seed with empty byte slices
	f.Add(EncodeBytesList([]byte{}, []byte{}, []byte{}))

	// Seed with binary data
	f.Add(EncodeBytesList([]byte{0x00, 0xff, 0xaa, 0x55}))

	// Seed with large data
	largeData := make([]byte, 1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	f.Add(EncodeBytesList(largeData))

	f.Fuzz(func(t *testing.T, data []byte) {
		// DecodeBytesList should never panic, even with arbitrary input
		result, err := DecodeBytesList(data)

		if err == nil {
			// If decoding succeeded, verify the result is reasonable
			for _, slice := range result {
				// Each slice should be non-nil (even if empty)
				if slice == nil {
					t.Fatal("DecodeBytesList returned nil slice in result")
				}
			}
		}
		// If err != nil, that's fine - malformed input should return an error, not panic
	})
}

// FuzzBinaryEncRoundtrip tests that encoding and then decoding produces the original data.
// This verifies the encode/decode cycle is lossless for any valid input.
func FuzzBinaryEncRoundtrip(f *testing.F) {
	// Seed with test cases from existing tests
	f.Add([]byte("foobar\x00baz"), []byte("\x00\x01\x00"), []byte("third"))

	// Seed with empty slices
	f.Add([]byte{}, []byte{}, []byte{})

	// Seed with single element
	f.Add([]byte("single"), []byte{}, []byte{})

	// Seed with binary data
	f.Add([]byte{0x00, 0xff}, []byte{0xaa, 0x55}, []byte{0x01, 0x02, 0x03})

	// Seed with varying lengths
	f.Add([]byte("a"), []byte("ab"), []byte("abc"))

	// Seed with special characters
	f.Add([]byte("\x00"), []byte("\xff"), []byte("\x00\xff"))

	// Seed with longer data
	f.Add([]byte("this is a longer test string"), []byte("another one"), []byte("and a third"))

	f.Fuzz(func(t *testing.T, slice1, slice2, slice3 []byte) {
		// Create input with three byte slices
		input := [][]byte{slice1, slice2, slice3}

		// Encode the input
		encoded := EncodeBytesList(input...)

		// Decode the result
		decoded, err := DecodeBytesList(encoded)
		if err != nil {
			t.Fatalf("DecodeBytesList failed on valid encoded data: %v", err)
		}

		// Verify we got the same number of slices back
		if len(decoded) != len(input) {
			t.Fatalf("DecodeBytesList returned %d slices, expected %d", len(decoded), len(input))
		}

		// Verify each slice matches the original
		for i := range input {
			if !bytes.Equal(decoded[i], input[i]) {
				t.Fatalf("Slice %d mismatch:\n  original: %v\n  decoded:  %v", i, input[i], decoded[i])
			}
		}

		// Additional verification: re-encoding should produce identical bytes
		reEncoded := EncodeBytesList(decoded...)
		if !bytes.Equal(encoded, reEncoded) {
			t.Fatal("Re-encoding decoded data produced different bytes")
		}
	})
}
