package env

import "math"

var (
	GraphQLQueryMaxDepth = &IntegerSetting{
		envVar:       "GRAPHQL_QUERY_MAX_DEPTH",
		defaultValue: 8,
		minimumValue: 8,
		maximumValue: math.MaxInt,
	}
)
