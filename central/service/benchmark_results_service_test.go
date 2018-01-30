package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func TestGroupPayloadsByScanID(t *testing.T) {

	scan1Time := ptypes.TimestampNow()
	scan1Time.Seconds -= 1000
	scan2Time := ptypes.TimestampNow()

	payloads := []*v1.BenchmarkResult{
		{
			ClusterId: "test",
			Host:      "roxbase",
			Id:        "id1",
			Results: []*v1.CheckResult{
				{
					Result: v1.CheckStatus_WARN,
					Notes: []string{
						"/var/lib/docker does not have its own partition",
					},
					Definition: &v1.CheckDefinition{
						Name:        "CIS Docker v1.1.0 - 1.1",
						Description: "Ensure a separate partition for containers has been created",
					},
				},
			},
			ScanId:  "scan1",
			EndTime: scan1Time,
		},
		{
			ClusterId: "test",
			Host:      "roxbase1",
			Id:        "id2",
			Results: []*v1.CheckResult{
				{
					Result: v1.CheckStatus_PASS,
					Definition: &v1.CheckDefinition{
						Name:        "CIS Docker v1.1.0 - 1.1",
						Description: "Ensure a separate partition for containers has been created",
					},
				},
			},
			ScanId:  "scan1",
			EndTime: scan1Time,
		},
		{
			ClusterId: "test",
			Host:      "roxbase1",
			Id:        "id3",
			Results: []*v1.CheckResult{
				{
					Notes: []string{
						"/var/lib/docker does not have its own partition",
					},
					Result: v1.CheckStatus_INFO,
					Definition: &v1.CheckDefinition{
						Name:        "CIS Docker v1.1.0 - 1.1",
						Description: "Ensure a separate partition for containers has been created",
					},
				},
			},
			ScanId:  "scan2",
			EndTime: scan2Time,
		},
	}

	expected := &v1.GetBenchmarkResultsGroupedResponse{
		Benchmarks: []*v1.BenchmarkResultsGrouped{
			{
				ScanId: "scan2",
				Time:   scan2Time,
				CheckResults: []*v1.BenchmarkResultsGrouped_ScopedCheckResult{
					{
						Definition: &v1.CheckDefinition{
							Name:        "CIS Docker v1.1.0 - 1.1",
							Description: "Ensure a separate partition for containers has been created",
						},
						AggregatedResults: map[string]int32{
							v1.CheckStatus_INFO.String(): 1,
						},
						HostResults: []*v1.BenchmarkResultsGrouped_ScopedCheckResult_HostResult{
							{
								Host: "roxbase1",
								Notes: []string{
									"/var/lib/docker does not have its own partition",
								},
								Result: v1.CheckStatus_INFO,
							},
						},
					},
				},
			},
			{
				ScanId: "scan1",
				Time:   scan1Time,
				CheckResults: []*v1.BenchmarkResultsGrouped_ScopedCheckResult{
					{
						Definition: &v1.CheckDefinition{
							Name:        "CIS Docker v1.1.0 - 1.1",
							Description: "Ensure a separate partition for containers has been created",
						},
						AggregatedResults: map[string]int32{
							v1.CheckStatus_PASS.String(): 1,
							v1.CheckStatus_WARN.String(): 1,
						},
						HostResults: []*v1.BenchmarkResultsGrouped_ScopedCheckResult_HostResult{
							{
								Host:   "roxbase",
								Result: v1.CheckStatus_WARN,
								Notes: []string{
									"/var/lib/docker does not have its own partition",
								},
							},
							{
								Host:   "roxbase1",
								Result: v1.CheckStatus_PASS,
							},
						},
					},
				},
			},
		},
	}
	resp := groupPayloadsByScanID(payloads)
	assert.Equal(t, expected, resp)
}
