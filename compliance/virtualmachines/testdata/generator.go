package testdata

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/protobuf/proto"
)

//go:generate go run ./cmd/generate --out-dir=.

const (
	defaultPackageKind = "binary"
	defaultArch        = "amd64"
)

// Options control how an IndexReport payload is generated.
type Options struct {
	VsockCID        uint32
	NumPackages     int
	NumRepositories int
	Randomize       bool
	Seed            int64
}

func (o Options) normalized() Options {
	if o.VsockCID == 0 {
		o.VsockCID = 100
	}
	if o.NumPackages <= 0 {
		o.NumPackages = 1
	}
	if o.NumRepositories <= 0 {
		o.NumRepositories = int(math.Max(1, math.Ceil(float64(o.NumPackages)/10.0)))
	}
	if o.Seed == 0 {
		if o.Randomize {
			o.Seed = time.Now().UnixNano()
		} else {
			o.Seed = 1
		}
	}
	return o
}

func (o Options) validate() error {
	if o.NumRepositories > o.NumPackages && o.NumPackages > 0 {
		return fmt.Errorf("numRepositories (%d) cannot exceed numPackages (%d)", o.NumRepositories, o.NumPackages)
	}
	return nil
}

// GenerateIndexReport returns a VM IndexReport populated according to Options.
func GenerateIndexReport(opts Options) (*v1.IndexReport, error) {
	opts = opts.normalized()
	if err := opts.validate(); err != nil {
		return nil, err
	}

	rng := rand.New(rand.NewSource(opts.Seed))
	moduleLen, packageDBLen := fieldLengths(opts.NumPackages)
	repositories := make(map[string]*v4.Repository, opts.NumRepositories)
	for i := 0; i < opts.NumRepositories; i++ {
		repoID := fmt.Sprintf("repo-%d", i)
		repositories[repoID] = &v4.Repository{
			Id:   repoID,
			Name: fmt.Sprintf("repository-%d", i),
			Uri:  fmt.Sprintf("https://repo%d.example.com", i),
			Key:  fmt.Sprintf("key-%d", i),
			Cpe:  generateValidCPE(i, opts.Randomize, rng),
		}
	}

	packages := make(map[string]*v4.Package, opts.NumPackages)
	for i := 0; i < opts.NumPackages; i++ {
		pkgID := fmt.Sprintf("pkg-%d", i)
		repoHint := ""
		if opts.NumRepositories > 0 {
			repoHint = fmt.Sprintf("repo-%d", i%opts.NumRepositories)
		}

		name := fmt.Sprintf("package-%d", i)
		version := fmt.Sprintf("1.%d.%d", i/10, i%10)
		module := paddedField("module", i, moduleLen)
		packageDB := paddedField("pkgdb", i, packageDBLen)
		cpe := generateValidCPE(i, opts.Randomize, rng)
		fixedIn := fmt.Sprintf("1.%d.%d", (i+5)/10, (i+7)%10)

		if opts.Randomize {
			name = fmt.Sprintf("%s-%d", name, rng.Intn(1000))
			version = fmt.Sprintf("1.%d.%d", rng.Intn(20), rng.Intn(100))
			module = paddedField(fmt.Sprintf("module-rand-%d", rng.Intn(1000)), i, moduleLen)
			packageDB = paddedField(fmt.Sprintf("pkgdb-rand-%d", rng.Intn(500)), i, packageDBLen)
			fixedIn = fmt.Sprintf("1.%d.%d", rng.Intn(20), rng.Intn(100))
		}

		normalizedVersion := &v4.NormalizedVersion{
			Kind: "semver",
			V: []int32{
				1,
				int32((i % 100) + 1),
				int32((i % 50) + 1),
			},
		}

		// Generate CPE for source package (using same index to keep it deterministic)
		srcCPE := generateValidCPE(i, opts.Randomize, rng)

		packages[pkgID] = &v4.Package{
			Id:                pkgID,
			Name:              name,
			Version:           version,
			NormalizedVersion: normalizedVersion,
			FixedInVersion:    fixedIn,
			Kind:              defaultPackageKind,
			PackageDb:         packageDB,
			Arch:              defaultArch,
			Module:            module,
			Cpe:               cpe,
			RepositoryHint:    repoHint,
			Source: &v4.Package{
				Id:      pkgID + "-src",
				Name:    name + "-src",
				Version: version,
				Kind:    defaultPackageKind,
				Arch:    defaultArch,
				Cpe:     srcCPE,
			},
		}
	}

	hashID := fmt.Sprintf("hash-vm-%d", opts.VsockCID)
	if opts.Randomize {
		hashID = fmt.Sprintf("%s-%d", hashID, rng.Intn(1_000_000))
	}

	report := &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", opts.VsockCID),
		IndexV4: &v4.IndexReport{
			HashId:  hashID,
			State:   "IndexFinished",
			Success: true,
			Contents: &v4.Contents{
				Packages:     packages,
				Repositories: repositories,
			},
		},
	}
	return report, nil
}

// SerializeReport marshals an IndexReport into protobuf bytes.
func SerializeReport(report *v1.IndexReport) ([]byte, error) {
	if report == nil {
		return nil, errors.New("report cannot be nil")
	}
	return proto.Marshal(report)
}

// WriteFixture creates/overwrites a protobuf file containing a generated IndexReport.
func WriteFixture(path string, opts Options) error {
	report, err := GenerateIndexReport(opts)
	if err != nil {
		return err
	}
	data, err := SerializeReport(report)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadFixture reads a protobuf IndexReport fixture from disk.
func LoadFixture(path string) (*v1.IndexReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	report := &v1.IndexReport{}
	if err := proto.Unmarshal(data, report); err != nil {
		return nil, err
	}
	return report, nil
}

func paddedField(prefix string, idx int, length int) string {
	if length <= 0 {
		return fmt.Sprintf("%s-%d", prefix, idx)
	}
	base := fmt.Sprintf("%s-%d-", prefix, idx)
	if len(base) >= length {
		return base[:length]
	}

	builder := strings.Builder{}
	builder.Grow(length)
	for builder.Len() < length {
		builder.WriteString(base)
		builder.WriteString("abcdefghijklmnopqrstuvwxyz0123456789")
	}
	result := builder.String()
	if len(result) > length {
		return result[:length]
	}
	return result
}

func fieldLengths(numPackages int) (moduleLen, packageDBLen int) {
	switch {
	case numPackages >= 1500:
		return 2880, 1440
	case numPackages >= 700:
		return 3072, 1536
	default:
		return 2560, 1280
	}
}

// generateValidCPE creates a valid CPE 2.3 formatted string for load testing.
// CPE format: cpe:2.3:part:vendor:product:version:update:edition:language:sw_edition:target_sw:target_hw:other
// All fields use '*' for ANY except vendor, product, and version which are populated with test values.
func generateValidCPE(idx int, randomize bool, rng *rand.Rand) string {
	vendor := fmt.Sprintf("vendor%d", idx%100)
	product := fmt.Sprintf("product%d", idx)
	version := fmt.Sprintf("1.%d.%d", idx/10, idx%10)

	if randomize {
		vendor = fmt.Sprintf("vendor%d", rng.Intn(1000))
		product = fmt.Sprintf("product%d", rng.Intn(10000))
		version = fmt.Sprintf("%d.%d.%d", rng.Intn(10), rng.Intn(100), rng.Intn(100))
	}

	// CPE 2.3 format with minimal fields populated
	// part=a (application), rest are wildcards (*)
	return fmt.Sprintf("cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*", vendor, product, version)
}
