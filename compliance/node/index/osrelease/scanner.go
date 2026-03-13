package osrelease

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/trace"
	"sort"
	"strings"

	"github.com/quay/claircore"
)

// Path and FallbackPath are the two documented locations for the os-release file.
const (
	Path         = `etc/os-release`
	FallbackPath = `usr/lib/os-release`
)

// NodeDistribution represents the distribution details extracted from os-release.
type NodeDistribution struct {
	ID               string
	DID              string
	Name             string
	Version          string
	VersionID        string
	VersionCodeName  string
	PrettyName       string
	PlatformID       string
	OpenShiftVersion string
}

type Scanner struct{}

// Scan reports the found os-release distribution information in the provided layer
// specifically returning the custom NodeDistribution type.
func (s *Scanner) Scan(ctx context.Context, l *claircore.Layer) (*NodeDistribution, error) {
	defer trace.StartRegion(ctx, "Scanner.Scan").End()
	slog.DebugContext(ctx, "start")
	defer slog.DebugContext(ctx, "done")

	sys, err := l.FS()
	if err != nil {
		return nil, fmt.Errorf("osrelease: unable to open layer: %w", err)
	}

	var rd io.Reader
	// Iterate through known paths
	for _, n := range []string{Path, FallbackPath} {
		f, err := sys.Open(n)
		if err != nil {
			slog.DebugContext(ctx, "unable to open file", "name", n, "reason", err)
			continue
		}
		// We read the content here because we need to close the file
		// before leaving the loop or returning.
		b, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("osrelease: failed to read %s: %w", n, err)
		}
		rd = bytes.NewReader(b)
		break
	}

	if rd == nil {
		slog.DebugContext(ctx, "didn't find an os-release file")
		return nil, nil
	}

	return toNodeDist(ctx, rd)
}

// toNodeDist handles the mapping from the raw map to our struct.
func toNodeDist(ctx context.Context, r io.Reader) (*NodeDistribution, error) {
	defer trace.StartRegion(ctx, "parse").End()
	m, err := Parse(ctx, r)
	if err != nil {
		return nil, err
	}

	d := &NodeDistribution{
		Name: "Linux",
		DID:  "linux",
	}

	// Sort keys for deterministic processing
	ks := make([]string, 0, len(m))
	for key := range m {
		ks = append(ks, key)
	}
	sort.Strings(ks)

	// Skip RHCOS CPE from os-release since it's not very useful
	for _, key := range ks {
		value := m[key]
		switch key {
		case "ID":
			d.ID = value
			d.DID = value
		case "VERSION_ID":
			d.VersionID = value
		case "NAME":
			d.Name = value
		case "VERSION":
			d.Version = value
		case "VERSION_CODENAME":
			d.VersionCodeName = value
		case "PRETTY_NAME":
			d.PrettyName = value
		case "PLATFORM_ID":
			d.PlatformID = value
		case "OPENSHIFT_VERSION":
			d.OpenShiftVersion = value
		case "REDHAT_BUGZILLA_PRODUCT":
			d.PrettyName = value
		}
	}

	// Dynamic CPE Generation for OpenShift
	if d.OpenShiftVersion != "" {
		d.CPE = guessOpenShiftCPE(d)
	}

	return d, nil
}

// Parse splits the contents of "r" into key-value pairs as described in os-release(5).
func Parse(ctx context.Context, r io.Reader) (map[string]string, error) {
	m := make(map[string]string)
	s := bufio.NewScanner(r)
	for s.Scan() && ctx.Err() == nil {
		b := bytes.TrimSpace(s.Bytes())
		if len(b) == 0 || b[0] == '#' {
			continue
		}
		if !bytes.ContainsRune(b, '=') {
			return nil, fmt.Errorf("osrelease: malformed line %q", s.Text())
		}
		key, value, _ := strings.Cut(string(b), "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case len(value) == 0:
		case value[0] == '\'':
			value = strings.TrimFunc(value, func(r rune) bool { return r == '\'' })
			value = strings.ReplaceAll(value, `'\''`, `'`)
		case value[0] == '"':
			value = strings.TrimFunc(value, func(r rune) bool { return r == '"' })
			value = dqReplacer.Replace(value)
		}
		m[key] = value
	}
	return m, s.Err()
}

var dqReplacer = strings.NewReplacer(
	"\\`", "`",
	`\\`, `\`,
	`\"`, `"`,
	`\$`, `$`,
)
