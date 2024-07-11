package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stretchr/testify/require"
)

func TestSchemaGeneration(t *testing.T) {
	s := resolvers.Schema()
	_, err := graphql.ParseSchema(s, resolvers.NewMock())

	searchKey := "line"
	if err != nil && strings.Contains(err.Error(), searchKey) {

		// derive line number from error string
		errString := err.Error()
		index := strings.LastIndex(errString, searchKey)
		var lineNum int
		_, scanErr := fmt.Sscan(errString[index+len(searchKey):], &lineNum)
		if scanErr != nil {
			// couldn't get line number, just fail out
			require.NoError(t, err, "Failed to parse GraphQL Schema")
		}

		// split the schema by newline
		lines := strings.SplitN(s, "\n", lineNum+1)
		if len(lines) <= lineNum {
			// couldn't get to the line, just fail out
			require.NoError(t, err, "Failed to parse GraphQL Schema")
		}

		// fail the test with the included failed line
		require.NoErrorf(t, err, "Failed to parse GraphQL Schema: %q", strings.TrimSpace(lines[lineNum-1]))
	}
	require.NoError(t, err, "Failed to parse GraphQL Schema")
}
