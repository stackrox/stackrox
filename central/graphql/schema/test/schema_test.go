package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCVSSFieldsAreNullable(t *testing.T) {
	schema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.NewMock())
	require.NoError(t, err)
	cases := map[string][]string{
		"ImageCVEV2":           {"cvss", "nvdcvss"},
		"ImageVulnerability":   {"cvss", "nvdCvss"},
		"NodeCVE":              {"cvss"},
		"NodeVulnerability":    {"cvss"},
		"ClusterCVE":           {"cvss"},
		"ClusterVulnerability": {"cvss"},
	}
	for name, fields := range cases {
		t.Run(name, func(t *testing.T) {
			response := schema.Exec(context.Background(), `query($name: String!) {
				__type(name: $name) { fields { name type { kind name } } }
			}`, "", map[string]interface{}{"name": name})
			require.Empty(t, response.Errors)
			var result struct {
				Type struct {
					Fields []struct {
						Name string
						Type struct{ Kind, Name string }
					}
				} `json:"__type"`
			}
			require.NoError(t, json.Unmarshal(response.Data, &result))
			got := make(map[string]string)
			for _, field := range result.Type.Fields {
				got[field.Name] = field.Type.Kind + ":" + field.Type.Name
			}
			for _, field := range fields {
				assert.Equal(t, "SCALAR:Float", got[field], "%s must be a nullable Float", field)
			}
		})
	}
}

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
