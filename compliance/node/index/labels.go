package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	// labelsJSONPaths are known locations for RH build labels.json on a node
	// root (or container layer). Matches Claircore RHCC detector paths with
	// the container "root/" prefix stripped for host mounts.
	labelsJSONPaths = []string{
		"usr/share/buildinfo/labels.json",
		"root/buildinfo/labels.json",
	}

	errLabelsNotFound = errors.New("labels.json not found")
)

// buildLabels holds identifying fields from labels.json.
// See Claircore rhel/rhcc labels schema.
type buildLabels struct {
	Created      time.Time `json:"org.opencontainers.image.created"`
	Architecture string    `json:"architecture"`
	Name         string    `json:"name"`
	CPE          string    `json:"cpe"`
}

// parseLabelsJSON reads the first available labels.json under hostPath.
// Missing files are not an error (returns errLabelsNotFound); other I/O or
// JSON errors are returned as-is.
func parseLabelsJSON(hostPath string) (buildLabels, string, error) {
	if hostPath == "" {
		return buildLabels{}, "", errLabelsNotFound
	}
	var lastNotExist error
	for _, relPath := range labelsJSONPaths {
		path := filepath.Join(hostPath, relPath)
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			var lb buildLabels
			if err := json.Unmarshal(data, &lb); err != nil {
				return buildLabels{}, "", fmt.Errorf("parsing labels.json %q: %w", path, err)
			}
			// Some images were published with an extra layer of JSON string
			// quoting on label values (same bug Claircore RHCC handles).
			for _, p := range []*string{&lb.Name, &lb.Architecture, &lb.CPE} {
				if u, err := strconv.Unquote(*p); err == nil {
					*p = u
				}
			}
			return lb, path, nil
		case errors.Is(err, os.ErrNotExist):
			lastNotExist = err
			continue
		default:
			return buildLabels{}, "", fmt.Errorf("opening labels.json %q: %w", path, err)
		}
	}
	if lastNotExist != nil {
		return buildLabels{}, "", errLabelsNotFound
	}
	return buildLabels{}, "", errLabelsNotFound
}
