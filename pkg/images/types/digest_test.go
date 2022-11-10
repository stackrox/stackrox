package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDigest_Digest(t *testing.T) {
	cases := []struct {
		sha      string
		expected string
	}{
		{
			sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		},
		{
			sha:      "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		},
		{
			sha:      "",
			expected: "",
		},
		{
			sha:      "sha512:267eebcd42de25e467db55ca95284244f95390c3c02da7b35c17ef3460aec60dc3a1a25e3ce00a9b18cf686ca4fef4429e88d1ac364e06cbd635381f489a9185",
			expected: "sha512:267eebcd42de25e467db55ca95284244f95390c3c02da7b35c17ef3460aec60dc3a1a25e3ce00a9b18cf686ca4fef4429e88d1ac364e06cbd635381f489a9185",
		},
	}
	for _, c := range cases {
		t.Run(c.sha, func(t *testing.T) {
			assert.Equal(t, c.expected, NewDigest(c.sha).Digest())
		})
	}
}

func TestDigest_Algorithm(t *testing.T) {
	cases := []struct {
		sha      string
		expected string
	}{
		{
			sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "sha256",
		},
		{
			sha:      "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "sha256",
		},
		{
			sha:      "",
			expected: "",
		},
		{
			sha:      "sha512:267eebcd42de25e467db55ca95284244f95390c3c02da7b35c17ef3460aec60dc3a1a25e3ce00a9b18cf686ca4fef4429e88d1ac364e06cbd635381f489a9185",
			expected: "sha512",
		},
	}
	for _, c := range cases {
		t.Run(c.sha, func(t *testing.T) {
			assert.Equal(t, c.expected, NewDigest(c.sha).Algorithm())
		})
	}
}

func TestDigest_Hash(t *testing.T) {
	cases := []struct {
		sha      string
		expected string
	}{
		{
			sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		},
		{
			sha:      "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		},
		{
			sha:      "",
			expected: "",
		},
		{
			sha:      "sha512:267eebcd42de25e467db55ca95284244f95390c3c02da7b35c17ef3460aec60dc3a1a25e3ce00a9b18cf686ca4fef4429e88d1ac364e06cbd635381f489a9185",
			expected: "267eebcd42de25e467db55ca95284244f95390c3c02da7b35c17ef3460aec60dc3a1a25e3ce00a9b18cf686ca4fef4429e88d1ac364e06cbd635381f489a9185",
		},
	}
	for _, c := range cases {
		t.Run(c.sha, func(t *testing.T) {
			assert.Equal(t, c.expected, NewDigest(c.sha).Hash())
		})
	}
}
