package inmem

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/toolkit/types"
)

// noMatch is the sentinel dict-id meaning "the record's value is not present in
// the fragment's dictionary", so no row can possibly match on that field.
const noMatch int32 = -1

// nameSpan identifies a contiguous range of row indices in byNamePostings.
type nameSpan struct {
	start int32
	count int32
}

// snapshot is an immutable, struct-of-arrays columnar fragment for a single
// updater. Per-row primitive columns and flat CSR byNamePostings are backed by
// an off-heap read-only mmap .col file, and cold display strings live in an
// off-heap zstd-compressed .blob file; string dictionaries and byNameSpan stay
// on the Go heap so returned Vulnerability values hold no pointers into mmap
// regions. Lifetime of both off-heap files is managed by atomic reference
// counting.
type snapshot struct {
	refs atomic.Int64
	id   uint32 // immutable, monotonic fragment ID assigned by Store

	// Dictionaries (dedup, Go heap). Row columns below index into these.
	cveNames  []string
	pkgNames  []string
	pkgVers   []string
	modules   []string
	arches    []string
	repoNames []string
	repoKeys  []string
	repoURIs  []string
	fixedVers []string
	verKinds  []string
	updaters  []string
	dists     []claircore.Distribution
	aliasDict []claircore.Alias

	// Known distributions contributed by this fragment (mirrors
	// scanner/datastore/postgres/distributions.go, including RHEL/Hummingbird CPEs).
	knownDists []claircore.Distribution

	// Per-row parallel columns (mmap-backed in cols after finish()).
	cveID      []int32
	pkgNameID  []int32
	pkgVerID   []int32
	kind       []uint8
	moduleID   []int32
	archID     []int32
	distID     []int16 // -1: no distribution
	repoNameID []int32
	repoKeyID  []int16
	repoURIID  []int32
	fixedID    []int32
	verKindID  []int8
	updaterID  []int32
	normSev    []uint8
	archop     []uint8
	invert     []uint8 // 0 or 1 (portable 1-byte layout for mmap backing)
	issued     []int64

	// Sparse vulnerable-range storage (only ~7.7% of rows have a range).
	rangeIdx []int32 // -1: no range; else index into rLower/rUpper
	rLower   [][10]int32
	rUpper   [][10]int32

	// Flat alias postings: row r's aliases are
	// aliasPostings[aliasStart[r]:aliasStart[r+1]].
	aliasStart    []int32 // len nrows+1
	aliasPostings []int32
	selfID        []int32 // -1: no self alias

	// Cold display (description/links/severity) off-heap dictionary indices.
	descI []int32
	linkI []int32
	sevI  []int32

	// Flat CSR package-name index (byNamePostings is mmap-backed in cols).
	byNamePostings []int32
	byNameSpan     map[string]nameSpan

	// Small query-time lookup dicts.
	moduleIdx    map[string]int32
	repoNameIdx  map[string]int32
	repoKeyIdx   map[string]int32
	fixedEmptyID int32

	display *displayStore
	cols    *colsStore

	nrows int
}

func (s *snapshot) retain() {
	if s != nil {
		s.refs.Add(1)
	}
}

func (s *snapshot) release() {
	if s != nil && s.refs.Add(-1) == 0 {
		if s.display != nil {
			s.display.release()
		}
		if s.cols != nil {
			s.cols.release()
		}
	}
}

func (s *snapshot) rowsForName(name string) []int32 {
	if name == "" {
		return nil
	}
	sp, ok := s.byNameSpan[name]
	if !ok || sp.count == 0 {
		return nil
	}
	return s.byNamePostings[sp.start : sp.start+sp.count]
}

// reconstruct rebuilds a *claircore.Vulnerability for a single row, excluding
// the cold display fields (description/links/severity), which are filled only
// for rows that survive opts.Vulnerable.
func (s *snapshot) reconstruct(row int32, id string) *claircore.Vulnerability {
	v := &claircore.Vulnerability{
		ID:                 id,
		Updater:            s.updaters[s.updaterID[row]],
		Name:               s.cveNames[s.cveID[row]],
		NormalizedSeverity: claircore.Severity(s.normSev[row]),
		FixedInVersion:     s.fixedVers[s.fixedID[row]],
		ArchOperation:      claircore.ArchOp(s.archop[row]),
		Invert:             s.invert[row] != 0,
		Issued:             time.Unix(s.issued[row], 0).UTC(),
		Package: &claircore.Package{
			Name:    s.pkgNames[s.pkgNameID[row]],
			Version: s.pkgVers[s.pkgVerID[row]],
			Kind:    types.PackageKind(s.kind[row]),
			Module:  s.modules[s.moduleID[row]],
			Arch:    s.arches[s.archID[row]],
		},
		Repo: &claircore.Repository{
			Name: s.repoNames[s.repoNameID[row]],
			Key:  s.repoKeys[s.repoKeyID[row]],
			URI:  s.repoURIs[s.repoURIID[row]],
		},
	}
	if d := s.distID[row]; d >= 0 {
		dd := s.dists[d]
		v.Dist = &dd
	} else {
		v.Dist = &claircore.Distribution{}
	}
	if ri := s.rangeIdx[row]; ri >= 0 {
		k := s.verKinds[s.verKindID[row]]
		v.Range = &claircore.Range{}
		v.Range.Lower.Kind, v.Range.Lower.V = k, s.rLower[ri]
		v.Range.Upper.Kind, v.Range.Upper.V = k, s.rUpper[ri]
	}
	if lo, hi := s.aliasStart[row], s.aliasStart[row+1]; hi > lo {
		v.Aliases = make([]claircore.Alias, 0, hi-lo)
		for _, p := range s.aliasPostings[lo:hi] {
			v.Aliases = append(v.Aliases, s.aliasDict[p])
		}
	}
	if sid := s.selfID[row]; sid >= 0 {
		v.Self = s.aliasDict[sid]
	}
	return v
}

func (s *snapshot) fillDisplay(v *claircore.Vulnerability, row int32) error {
	if s.display == nil {
		return nil
	}
	var err error
	if v.Description, err = s.display.description(s.descI[row]); err != nil {
		return err
	}
	if v.Links, err = s.display.links(s.linkI[row]); err != nil {
		return err
	}
	if v.Severity, err = s.display.severity(s.sevI[row]); err != nil {
		return err
	}
	return nil
}

// decodeCache memoizes decompressed display strings for the span of a single
// Get call. Display is heavily deduplicated across rows, so caching turns
// O(matches) zstd decompressions into O(distinct strings).
type decodeCache struct {
	m map[dcKey]string
}

type dcKey struct {
	fragID uint32
	field  uint8
	idx    int32
}

func newDecodeCache() *decodeCache { return &decodeCache{m: make(map[dcKey]string)} }

func (c *decodeCache) get(fragID uint32, field uint8, idx int32, decode func(int32) (string, error)) (string, error) {
	k := dcKey{fragID: fragID, field: field, idx: idx}
	if s, ok := c.m[k]; ok {
		return s, nil
	}
	s, err := decode(idx)
	if err != nil {
		return "", err
	}
	c.m[k] = s
	return s, nil
}

func (s *snapshot) fillDisplayCached(v *claircore.Vulnerability, row int32, cache *decodeCache) error {
	if s.display == nil {
		return nil
	}
	if cache == nil {
		return s.fillDisplay(v, row)
	}
	var err error
	if v.Description, err = cache.get(s.id, 0, s.descI[row], s.display.description); err != nil {
		return err
	}
	if v.Links, err = cache.get(s.id, 1, s.linkI[row], s.display.links); err != nil {
		return err
	}
	if v.Severity, err = cache.get(s.id, 2, s.sevI[row], s.display.severity); err != nil {
		return err
	}
	return nil
}

// recFilter precomputes, once per index record, acceptable dictionary IDs and a
// per-distribution boolean mask so the per-row test is pure integer/boolean
// comparison.
type recFilter struct {
	hasModule, hasRepoName, hasRepoKey, hasFixed bool
	module, repoName, repoKey                    int32

	distConstrained bool
	okDist          []bool
	okEmptyDist     bool

	versionFilter bool
	recV          [10]int32
	recVKind      string
}

func (s *snapshot) buildFilter(rec *claircore.IndexRecord, opts datastore.GetOpts) (recFilter, error) {
	f := recFilter{module: noMatch, repoName: noMatch, repoKey: noMatch}
	lk := func(m map[string]int32, key string) int32 {
		if id, ok := m[key]; ok {
			return id
		}
		return noMatch
	}
	var distMs []driver.MatchConstraint
	for _, m := range opts.Matchers {
		switch m {
		case driver.PackageModule:
			f.hasModule = true
			if rec.Package != nil {
				f.module = lk(s.moduleIdx, rec.Package.Module)
			}
		case driver.RepositoryName:
			f.hasRepoName = true
			if rec.Repository != nil {
				f.repoName = lk(s.repoNameIdx, rec.Repository.Name)
			}
		case driver.RepositoryKey:
			f.hasRepoKey = true
			if rec.Repository != nil {
				f.repoKey = lk(s.repoKeyIdx, rec.Repository.Key)
			}
		case driver.HasFixedInVersion:
			f.hasFixed = true
		case driver.DistributionDID, driver.DistributionName, driver.DistributionVersionID,
			driver.DistributionVersion, driver.DistributionVersionCodeName,
			driver.DistributionPrettyName, driver.DistributionCPE, driver.DistributionArch:
			distMs = append(distMs, m)
		default:
			return recFilter{}, fmt.Errorf("inmem: unknown match constraint: %v", m)
		}
	}
	if len(distMs) > 0 {
		f.distConstrained = true
		f.okDist = make([]bool, len(s.dists))
		for i := range s.dists {
			f.okDist[i] = distMatches(s.dists[i], rec.Distribution, distMs)
		}
		f.okEmptyDist = distMatches(claircore.Distribution{}, rec.Distribution, distMs)
	}
	if opts.VersionFiltering && rec.Package != nil {
		f.versionFilter = true
		f.recV = rec.Package.NormalizedVersion.V
		f.recVKind = rec.Package.NormalizedVersion.Kind
	}
	return f, nil
}

func distMatches(vd claircore.Distribution, rd *claircore.Distribution, ms []driver.MatchConstraint) bool {
	if rd == nil {
		return false
	}
	for _, m := range ms {
		switch m {
		case driver.DistributionDID:
			if vd.DID != rd.DID {
				return false
			}
		case driver.DistributionName:
			if vd.Name != rd.Name {
				return false
			}
		case driver.DistributionVersionID:
			if vd.VersionID != rd.VersionID {
				return false
			}
		case driver.DistributionVersion:
			if vd.Version != rd.Version {
				return false
			}
		case driver.DistributionVersionCodeName:
			if vd.VersionCodeName != rd.VersionCodeName {
				return false
			}
		case driver.DistributionPrettyName:
			if vd.PrettyName != rd.PrettyName {
				return false
			}
		case driver.DistributionArch:
			if vd.Arch != rd.Arch {
				return false
			}
		case driver.DistributionCPE:
			if vd.CPE.String() != rd.CPE.String() {
				return false
			}
		}
	}
	return true
}

// rowPasses is the per-row, allocation-free predicate replicating Postgres
// querybuilder.go's WHERE clause.
func (s *snapshot) rowPasses(row int32, f *recFilter) bool {
	if f.hasModule {
		if f.module == noMatch || s.moduleID[row] != f.module {
			return false
		}
	}
	if f.hasRepoName {
		if f.repoName == noMatch || s.repoNameID[row] != f.repoName {
			return false
		}
	}
	if f.hasRepoKey {
		if f.repoKey == noMatch || int32(s.repoKeyID[row]) != f.repoKey {
			return false
		}
	}
	if f.hasFixed {
		if s.fixedID[row] == s.fixedEmptyID {
			return false
		}
	}
	if f.distConstrained {
		if d := s.distID[row]; d >= 0 {
			if !f.okDist[d] {
				return false
			}
		} else if !f.okEmptyDist {
			return false
		}
	}
	if f.versionFilter {
		ri := s.rangeIdx[row]
		if ri < 0 || s.verKinds[s.verKindID[row]] != f.recVKind {
			return false
		}
		if !(cmp10(f.recV, s.rLower[ri]) >= 0 && cmp10(f.recV, s.rUpper[ri]) < 0) {
			return false
		}
	}
	return true
}

// matchRecord evaluates all candidate rows for rec in this fragment, invokes
// opts.Vulnerable (when provided) BEFORE decompressing cold display strings,
// and forwards surviving vulnerabilities to emit.
func (s *snapshot) matchRecord(
	ctx context.Context,
	rec *claircore.IndexRecord,
	opts datastore.GetOpts,
	cache *decodeCache,
	emit func(*claircore.Vulnerability),
) error {
	pkgRows := s.rowsForName(rec.Package.Name)
	var srcRows []int32
	if rec.Package.Source != nil {
		srcRows = s.rowsForName(rec.Package.Source.Name)
	}
	if len(pkgRows) == 0 && len(srcRows) == 0 {
		return nil
	}

	f, err := s.buildFilter(rec, opts)
	if err != nil {
		return err
	}
	var seen map[int32]struct{}
	if len(pkgRows) > 0 && len(srcRows) > 0 {
		seen = make(map[int32]struct{})
	}

	consider := func(rows []int32, kind types.PackageKind) error {
		for _, row := range rows {
			if s.kind[row] != uint8(kind) {
				continue
			}
			if seen != nil {
				if _, dup := seen[row]; dup {
					continue
				}
				seen[row] = struct{}{}
			}
			if !s.rowPasses(row, &f) {
				continue
			}
			v := s.reconstruct(row, globalID(s.id, row))
			if opts.Vulnerable != nil {
				ok, err := opts.Vulnerable(ctx, rec, v)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
			}
			if err := s.fillDisplayCached(v, row, cache); err != nil {
				return err
			}
			emit(v)
		}
		return nil
	}

	if err := consider(pkgRows, rec.Package.Kind); err != nil {
		return err
	}
	if rec.Package.Source != nil && len(srcRows) > 0 {
		if err := consider(srcRows, rec.Package.Source.Kind); err != nil {
			return err
		}
	}
	return nil
}

// cmp10 compares two [10]int32 vectors lexicographically, matching Postgres
// int[] ordering used by vulnerable_range @>.
func cmp10(a, b [10]int32) int {
	for i := range 10 {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}
