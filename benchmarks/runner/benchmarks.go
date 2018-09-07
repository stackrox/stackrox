package runner

import (
	"fmt"
	"os"
	"strings"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

// RunBenchmark runs a benchmark based on environment variables
func RunBenchmark() *v1.BenchmarkResult {
	hostname, err := getHostname()
	if err != nil {
		log.Fatalf("Could not find this node's hostname: %+v", err)
	}
	protoStartTime := ptypes.TimestampNow()
	checkResults := runBenchmark()
	protoEndTime := ptypes.TimestampNow()
	result := &v1.BenchmarkResult{
		Id:          uuid.NewV4().String(),
		Results:     checkResults,
		StartTime:   protoStartTime,
		EndTime:     protoEndTime,
		Host:        hostname,
		ScanId:      env.ScanID.Setting(),
		BenchmarkId: env.BenchmarkID.Setting(),
		Reason:      v1.BenchmarkReason(v1.BenchmarkReason_value[env.BenchmarkReason.Setting()]),
	}
	return result
}

func runBenchmark() []*v1.CheckResult {
	checks := renderChecks()

	results := make([]*v1.CheckResult, 0, len(checks))
Loop:
	for _, check := range checks {
		definition := check.Definition().CheckDefinition
		for _, dep := range check.Definition().Dependencies {
			if err := dep(); err != nil {
				msg := fmt.Sprintf("Skipping Test %v due to err in dependency: %+v", check.Definition().Name, err)
				result := &v1.CheckResult{
					Definition: &definition,
					Result:     v1.CheckStatus_NOTE,
					Notes:      []string{msg},
				}
				results = append(results, result)
				continue Loop
			}
		}
		result := check.Run()
		result.Definition = &definition
		results = append(results, &result)
	}
	return results
}

func getHostname() (string, error) {
	if err := os.Setenv("DOCKER_HOST", "unix://"+utils.ContainerPathPrefix+"/var/run/docker.sock"); err != nil {
		log.Fatalf("Unable to set DOCKER_HOST: %+v", err)
	}
	cli, err := docker.NewClient()
	if err != nil {
		return "", fmt.Errorf("docker client setup: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := cli.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("docker info: %s", err)
	}
	return info.Name, nil
}

func renderChecks() []utils.Check {
	checkStrs := strings.Split(env.Checks.Setting(), ",")
	var benchmarkChecks []utils.Check
	for _, checkStr := range checkStrs {
		check, ok := checks.Registry[checkStr]
		if !ok {
			log.Errorf("Check %v is not currently supported. Supported checks are %+v", checkStr, checks.Registry)
			continue
		}
		benchmarkChecks = append(benchmarkChecks, check)
	}
	return benchmarkChecks
}
