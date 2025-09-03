package rpm

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// CollectPackages executes rpm -qa and parses the output into PackageInfo structs
func CollectPackages() ([]PackageInfo, error) {
	// Execute rpm command with the same format as the Rust implementation
	cmd := exec.Command("rpm", "-qa", "--qf", "%{NAME}|%{VERSION}|%{RELEASE}|%{ARCH}\n")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute rpm command: %w", err)
	}

	return parseRPMOutput(string(output))
}

// parseRPMOutput parses the rpm -qa output into PackageInfo structs
func parseRPMOutput(output string) ([]PackageInfo, error) {
	var packages []PackageInfo

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			log.Warnf("Skipping malformed RPM output line: %s", line)
			continue
		}

		pkg := PackageInfo{
			Name:    parts[0],
			Version: parts[1],
			Release: parts[2],
			Arch:    parts[3],
		}

		packages = append(packages, pkg)
	}

	if len(packages) == 0 {
		return nil, errors.New("no packages found in rpm output")
	}

	return packages, nil
}
