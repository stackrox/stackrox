package parser

import (
	"net/url"
	"testing"
)

func FuzzParseURLQuery(f *testing.F) {
	// Seed with valid examples from existing tests
	f.Add("query=Namespace:ABC&pagination.offset=5&pagination.limit=50&pagination.sortOption.field=Deployment&pagination.sortOption.reversed=true")
	f.Add("query=Namespace:ABC+Cluster:ABC&pagination.offset=5&pagination.limit=50")
	f.Add("query=Deployment:nginx&pagination.limit=10")
	f.Add("query=Image+Tag:latest")
	f.Add("query=Deployment:nginx")
	f.Add("pagination.limit=100")
	f.Add("query=")
	f.Add("")

	// Edge cases with special characters
	f.Add("query=Deployment:nginx&query=Image:alpine")
	f.Add("query=Name:test%20value")
	f.Add("query=CVE:CVE-2021-44228")
	f.Add("pagination.offset=0&pagination.limit=0")
	f.Add("pagination.sortOption.reversed=false")

	// Complex queries
	f.Add("query=Namespace:kube-system+Deployment:coredns&pagination.limit=25&pagination.offset=10")
	f.Add("query=Image+Tag:v1.2.3+Namespace:default")

	// Invalid but parseable values
	f.Add("pagination.offset=-1")
	f.Add("pagination.limit=999999999")
	f.Add("pagination.sortOption.field=")
	f.Add("blah=blah&foo=bar")

	f.Fuzz(func(t *testing.T, rawQuery string) {
		// Parse the raw query string into url.Values
		values, err := url.ParseQuery(rawQuery)
		if err != nil {
			// If url.ParseQuery fails, skip this input as it's invalid at a lower level
			t.Skip()
		}

		// The test succeeds if ParseURLQuery doesn't panic
		// We don't assert on the output since the function may legitimately return errors
		// for invalid input, but it should never panic
		_, _, _ = ParseURLQuery(values)
	})
}
