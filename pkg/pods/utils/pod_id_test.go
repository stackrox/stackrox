package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestPodIDToString(t *testing.T) {
	p := PodID{
		Name:      "mypod",
		Namespace: "myns",
		UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
	}

	expected := "mypod.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066"

	assert.Equal(t, expected, p.String())
}

func TestPodIDWithPeriodToString(t *testing.T) {
	p := PodID{
		Name:      "my.pod",
		Namespace: "myns",
		UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
	}

	expected := "my.pod.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066"

	assert.Equal(t, expected, p.String())
}

func TestParsePodID(t *testing.T) {
	validCases := map[string]struct {
		input    string
		wantName string
		wantNS   string
		wantUID  string
	}{
		"simple": {
			input:    "mypod.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066",
			wantName: "mypod", wantNS: "myns", wantUID: "ebf487f0-a7c3-11e8-8600-42010a8a0066",
		},
		"dot in name": {
			input:    "my-po.d.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066",
			wantName: "my-po.d", wantNS: "myns", wantUID: "ebf487f0-a7c3-11e8-8600-42010a8a0066",
		},
		"minimal": {
			input: "a.b@c", wantName: "a", wantNS: "b", wantUID: "c",
		},
		"uppercase hex in uid": {
			input: "a.b@ABCDEF", wantName: "a", wantNS: "b", wantUID: "ABCDEF",
		},
		"hyphens everywhere": {
			input: "pod-name.ns-name@abc-def", wantName: "pod-name", wantNS: "ns-name", wantUID: "abc-def",
		},
		"multiple dots in name": {
			input: "a.b.c.d.ns@abc", wantName: "a.b.c.d", wantNS: "ns", wantUID: "abc",
		},
		"numeric parts": {
			input: "1.2@a", wantName: "1", wantNS: "2", wantUID: "a",
		},
		"consecutive hyphens in name": {
			input: "po--d.ns@abc", wantName: "po--d", wantNS: "ns", wantUID: "abc",
		},
		"consecutive hyphens in namespace": {
			input: "pod.n--s@abc", wantName: "pod", wantNS: "n--s", wantUID: "abc",
		},
		"double dot in name": {
			input: "a..b.ns@abc", wantName: "a..b", wantNS: "ns", wantUID: "abc",
		},
		"hyphen-dot in name": {
			input: "po-.d.ns@abc", wantName: "po-.d", wantNS: "ns", wantUID: "abc",
		},
		"dot-hyphen in name": {
			input: "po.-d.ns@abc", wantName: "po.-d", wantNS: "ns", wantUID: "abc",
		},
		"uid all hyphens": {
			input: "pod.ns@---", wantName: "pod", wantNS: "ns", wantUID: "---",
		},
		"uid single char": {
			input: "pod.ns@a", wantName: "pod", wantNS: "ns", wantUID: "a",
		},
		"uid single hyphen": {
			input: "a.b@-", wantName: "a", wantNS: "b", wantUID: "-",
		},
		"mixed case uid": {
			input: "a.b@aAbBcC", wantName: "a", wantNS: "b", wantUID: "aAbBcC",
		},
	}

	for name, tc := range validCases {
		t.Run(name, func(t *testing.T) {
			got, err := ParsePodID(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, got.Name)
			assert.Equal(t, tc.wantNS, got.Namespace)
			assert.Equal(t, tc.wantUID, string(got.UID))
		})
	}

	invalidCases := map[string]string{
		"no dot":                  "mypodwithoutns@ebf487f0-a7c3-11e8-8600-42010a8a0066",
		"leading dot":             ".mypodwithoutns@ebf487f0-a7c3-11e8-8600-42010a8a0066",
		"name ends with hyphen":   "pod-.ns@abc",
		"ns starts with hyphen":   "pod.-ns@abc",
		"ns ends with hyphen":     "pod.ns-@abc",
		"name starts with hyphen": "-pod.ns@abc",
		"empty uid":               "pod.ns@",
		"empty namespace":         "pod.@abc",
		"dot-at only":             ".@abc",
		"no name or namespace":    "@abc",
		"empty string":            "",
		"no at-sign":              "pod.ns",
		"uppercase in name":       "Pod.ns@abc",
		"uppercase in namespace":  "pod.NS@abc",
		"extra at-sign":           "pod.ns@abc@extra",
		"non-hex in uid g":        "pod.ns@g",
		"non-hex in uid xyz":      "pod.ns@xyz",
		"non-hex in uid GGG":      "pod.ns@GGG",
		"double dot before ns":    "pod..ns@abc",
		"trailing dot in name":    "pod.name..ns@abc",
		"space in name":           "pod .ns@abc",
		"space in uid":            "pod.ns@ab c",
	}

	for name, input := range invalidCases {
		t.Run(name, func(t *testing.T) {
			_, err := ParsePodID(input)
			assert.Error(t, err, "input: %q", input)
		})
	}
}

func TestParsePodIDRoundTrip(t *testing.T) {
	original := PodID{
		Name:      "my-pod.name",
		Namespace: "my-ns",
		UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
	}
	parsed, err := ParsePodID(original.String())
	require.NoError(t, err)
	assert.Equal(t, original, parsed)
}
