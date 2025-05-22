// Package common provides utilities for aggregating findings into Prometheus metrics.
// It includes functionality for parsing metric configurations, matching findings
// against label expressions, and generating aggregation results.
//
// # Finding Aggregation
//
// Finding aggregation is the process of grouping findings based on specific label
// expressions and generating metrics that summarize the findings. The aggregation
// process involves the following steps:
//
//  1. **Matching Labels**: The `collectMatchingLabels` function iterates over the label
//     expressions and evaluates whether a finding matches the specified conditions.
//     It yields the labels and their corresponding values that satisfy the expressions.
//
//  2. **Generating Aggregation Keys**: The `makeAggregationKey` function
//     computes a unique `aggregationKey` for each set of matching labels. The `aggregationKey`
//     is a string representation of the label values, sorted according to a predefined
//     label order.
//
//  3. **Storing Aggregation Results**: The aggregation results are computed by the `count`
//     method of the `aggregator` structure. The result is stored in the `result` property
//     which maps metric names to their corresponding records. Each record contains
//     the labels and the total count of findings that match the aggregation criteria.
//
// # Aggregation Key
//
// An `aggregationKey` is a unique identifier for a set of label values. It is constructed
// by concatenating the values of the labels in a predefined order, separated by a
// delimiter (`|`). This ensures that each combination of label values has a unique
// key.
//
// Example:
//
// Given the following label expressions and finding:
//
// Label Expressions:
//   - "Cluster": `=*prod`
//   - "Deployment": `=*backend`
//
// Finding:
//   - "Cluster": "pre-prod"
//   - "Deployment": "backend"
//
// The resulting `aggregationKey` would be:
//
//	"pre-prod|backend"
//
// The corresponding Prometheus labels would be:
//
//	{"Cluster": "pre-prod", "Deployment": "backend"}
//
// This key is used to uniquely identify and aggregate findings that match the same
// set of label expressions.
package common
