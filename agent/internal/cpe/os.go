package cpe

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/stackrox/rox/pkg/logging"
)

// OSInfo contains operating system information for CPE generation
type OSInfo struct {
	ID         string // e.g., "rhel", "fedora"
	Name       string // e.g., "Red Hat Enterprise Linux"
	Version    string // e.g., "9.4"
	VersionID  string // e.g., "9"
	PrettyName string // e.g., "Red Hat Enterprise Linux 9.4 (Plow)"
	CPEName    string // System CPE in 2.3 format
	Arch       string // System architecture
}

// ParseOSRelease parses /etc/os-release and returns OS information
func ParseOSRelease() (*OSInfo, error) {
	osRelease, err := parseOSReleaseFile("/etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("failed to parse /etc/os-release: %w", err)
	}

	// Get system architecture
	arch, err := getSystemArch()
	if err != nil {
		return nil, fmt.Errorf("failed to get system architecture: %w", err)
	}

	// Convert CPE from 2.2 to 2.3 format if needed
	systemCPE := ""
	if cpeName, exists := osRelease["CPE_NAME"]; exists {
		systemCPE = convertCPE22to23(cpeName)
	}

	return &OSInfo{
		ID:         osRelease["ID"],
		Name:       osRelease["NAME"],
		Version:    osRelease["VERSION_ID"],
		VersionID:  strings.Split(osRelease["VERSION_ID"], ".")[0], // Major version only
		PrettyName: osRelease["PRETTY_NAME"],
		CPEName:    systemCPE,
		Arch:       arch,
	}, nil
}

// parseOSReleaseFile parses an os-release format file
func parseOSReleaseFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error but don't override the main error
			log := logging.LoggerForModule()
			log.Warnf("Failed to close file %s: %v", filename, closeErr)
		}
	}()

	fields := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], `"`)
		fields[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fields, fmt.Errorf("error reading file: %w", err)
	}
	return fields, nil
}

// getSystemArch returns the system architecture using uname -m
func getSystemArch() (string, error) {
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// convertCPE22to23 converts CPE 2.2 format to CPE 2.3 format
// Input: cpe:/part:vendor:product:version
// Output: cpe:2.3:part:vendor:product:version:*:*:*:*:*:*:*
func convertCPE22to23(cpe22 string) string {
	if !strings.HasPrefix(cpe22, "cpe:/") {
		return cpe22 // Return as-is if not CPE 2.2 format
	}

	// Remove cpe:/ prefix and split by colons
	parts := strings.Split(strings.TrimPrefix(cpe22, "cpe:/"), ":")
	if len(parts) < 4 {
		return cpe22 // Return as-is if malformed
	}

	// CPE 2.3 format with wildcards for unspecified fields
	return fmt.Sprintf("cpe:2.3:%s:%s:%s:%s:*:*:*:*:*:*:*",
		parts[0], // part (o, a, h)
		parts[1], // vendor
		parts[2], // product
		parts[3], // version
	)
}
