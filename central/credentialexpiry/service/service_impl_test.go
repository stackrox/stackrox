package service

import (
	"testing"

	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestEnsureTLSAndReturnAddr(t *testing.T) {
	for _, testCase := range []struct {
		endpoint    string
		expectedOut string
		errExpected bool
	}{
		{
			endpoint: "scanner.stackrox", errExpected: true,
		},
		{
			endpoint: "http://scanner.stackrox", errExpected: true,
		},
		{
			endpoint: "https://scanner.stackrox", expectedOut: "scanner.stackrox:443",
		},
		{
			endpoint: "https://scanner.stackrox:8080", expectedOut: "scanner.stackrox:8080",
		},
		{
			endpoint: "https://scanner.stackrox/", expectedOut: "scanner.stackrox:443",
		},
		{
			endpoint: "https://scanner.stackrox/ping", expectedOut: "scanner.stackrox:443",
		},
		{
			endpoint: "https://scanner.stackrox:8080/", expectedOut: "scanner.stackrox:8080",
		},
		{
			endpoint: "https://scanner.stackrox:8080/ping", expectedOut: "scanner.stackrox:8080",
		},
	} {
		c := testCase
		t.Run(c.endpoint, func(t *testing.T) {
			got, err := ensureTLSAndReturnAddr(c.endpoint)
			if c.errExpected {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, c.expectedOut, got)
		})
	}
}
