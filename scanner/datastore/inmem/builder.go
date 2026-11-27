package inmem

import (
	"context"
	"fmt"
	"math"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
)

// snapshotBuilder incrementally compiles an immutable columnar fragment from a
// vulnerability stream. Display strings are streamed directly to the off-heap
// blob as rows arrive so the raw feed is never retained in memory.
type snapshotBuilder struct {
	s       *snapshot
	dir     string
	maxRows int

	cve     *intern[string]
	pkg     *intern[string]
	pkgVer  *intern[string]
	mod     *intern[string]
	arch    *intern[string]
	rn      *intern[string]
	rk      *intern[string]
	ru      *intern[string]
	fx      *intern[string]
	vk      *intern[string]
	upd     *intern[string]
	aliasIx map[claircore.Alias]int32

	distIx     map[string]int16
	knownDists map[claircore.Distribution]struct{}
	byName     map[string][]int32
	display    *displayBuilder

	row int32
}

func newSnapshotBuilder(dir string, fragID uint32, maxRows int) (*snapshotBuilder, error) {
	disp, err := newDisplayBuilder(dir)
	if err != nil {
		return nil, err
	}
	return &snapshotBuilder{
		s: &snapshot{
			id:         fragID,
			aliasStart: []int32{0},
		},
		dir:        dir,
		maxRows:    maxRows,
		cve:        newIntern[string](),
		pkg:        newIntern[string](),
		pkgVer:     newIntern[string](),
		mod:        newIntern[string](),
		arch:       newIntern[string](),
		rn:         newIntern[string](),
		rk:         newIntern[string](),
		ru:         newIntern[string](),
		fx:         newIntern[string](),
		vk:         newIntern[string](),
		upd:        newIntern[string](),
		aliasIx:    make(map[claircore.Alias]int32),
		distIx:     make(map[string]int16),
		knownDists: make(map[claircore.Distribution]struct{}),
		byName:     make(map[string][]int32),
		display:    disp,
	}, nil
}

func (b *snapshotBuilder) abort() {
	if b.display != nil {
		b.display.abort()
	}
}

// skip mirrors claircore datastore/postgres skipVulnerability: rows without a
// package name are never stored and therefore never matched.
func skip(v *claircore.Vulnerability) bool {
	return v == nil || v.Package == nil || v.Package.Name == ""
}

func (b *snapshotBuilder) add(v *claircore.Vulnerability) error {
	if skip(v) {
		return nil
	}
	if b.row == math.MaxInt32 {
		return fmt.Errorf("inmem: fragment row count overflow (%d)", math.MaxInt32)
	}
	if b.maxRows > 0 && int(b.row) >= b.maxRows {
		return fmt.Errorf("inmem: fragment exceeds max_rows limit (%d)", b.maxRows)
	}

	s := b.s
	pkg := v.Package

	cveID, _ := b.cve.id(v.Name)
	pkgNameID, _ := b.pkg.id(pkg.Name)
	pkgVerID, _ := b.pkgVer.id(pkg.Version)
	moduleID, _ := b.mod.id(pkg.Module)
	archID, _ := b.arch.id(pkg.Arch)
	// Store v.Updater verbatim to match ClairCore Postgres updatevulnerabilities.go.
	updID, _ := b.upd.id(v.Updater)

	var repoName, repoKey, repoURI string
	if v.Repo != nil {
		repoName, repoKey, repoURI = v.Repo.Name, v.Repo.Key, v.Repo.URI
	}
	rnID, _ := b.rn.id(repoName)
	rkID, _ := b.rk.id(repoKey)
	if rkID > math.MaxInt16 {
		return fmt.Errorf("inmem: repoKey dictionary overflow (%d > %d)", rkID, math.MaxInt16)
	}
	ruID, _ := b.ru.id(repoURI)
	fixedID, _ := b.fx.id(v.FixedInVersion)

	did, err := b.distID(v.Dist)
	if err != nil {
		return err
	}
	if kd, ok := extractKnownDist(v.Dist, repoName); ok {
		b.knownDists[kd] = struct{}{}
	}

	// Version range (sparse). A range is usable only when both bounds share a
	// non-empty kind, matching datastore/postgres rangekind.
	verKind := ""
	hasRange := v.Range != nil && v.Range.Lower.Kind != "" && v.Range.Lower.Kind == v.Range.Upper.Kind
	if hasRange {
		verKind = v.Range.Lower.Kind
	}
	vkID, _ := b.vk.id(verKind)
	if vkID > math.MaxInt8 {
		return fmt.Errorf("inmem: verKind dictionary overflow (%d > %d)", vkID, math.MaxInt8)
	}

	// Cold display off-heap.
	di, li, si, err := b.display.write(v.Description, v.Links, v.Severity)
	if err != nil {
		return err
	}

	s.cveID = append(s.cveID, cveID)
	s.pkgNameID = append(s.pkgNameID, pkgNameID)
	s.pkgVerID = append(s.pkgVerID, pkgVerID)
	s.kind = append(s.kind, uint8(pkg.Kind))
	s.moduleID = append(s.moduleID, moduleID)
	s.archID = append(s.archID, archID)
	s.repoNameID = append(s.repoNameID, rnID)
	s.repoKeyID = append(s.repoKeyID, int16(rkID))
	s.repoURIID = append(s.repoURIID, ruID)
	s.fixedID = append(s.fixedID, fixedID)
	s.updaterID = append(s.updaterID, updID)
	s.normSev = append(s.normSev, uint8(v.NormalizedSeverity))
	s.archop = append(s.archop, uint8(v.ArchOperation))
	var inv uint8
	if v.Invert {
		inv = 1
	}
	s.invert = append(s.invert, inv)
	s.issued = append(s.issued, v.Issued.Unix())
	s.distID = append(s.distID, did)
	s.verKindID = append(s.verKindID, int8(vkID))
	if hasRange {
		s.rangeIdx = append(s.rangeIdx, int32(len(s.rLower)))
		s.rLower = append(s.rLower, v.Range.Lower.V)
		s.rUpper = append(s.rUpper, v.Range.Upper.V)
	} else {
		s.rangeIdx = append(s.rangeIdx, noMatch)
	}
	s.descI = append(s.descI, di)
	s.linkI = append(s.linkI, li)
	s.sevI = append(s.sevI, si)

	// Match Postgres updatevulnerabilities.go: skip invalid aliases.
	for _, a := range v.Aliases {
		if a.Valid() {
			s.aliasPostings = append(s.aliasPostings, b.aliasID(a))
		}
	}
	s.aliasStart = append(s.aliasStart, int32(len(s.aliasPostings)))
	if v.Self.Valid() {
		s.selfID = append(s.selfID, b.aliasID(v.Self))
	} else {
		s.selfID = append(s.selfID, noMatch)
	}

	b.byName[pkg.Name] = append(b.byName[pkg.Name], b.row)
	b.row++
	return nil
}

func (b *snapshotBuilder) aliasID(a claircore.Alias) int32 {
	if id, ok := b.aliasIx[a]; ok {
		return id
	}
	id := int32(len(b.s.aliasDict))
	b.aliasIx[a] = id
	b.s.aliasDict = append(b.s.aliasDict, a)
	return id
}

func (b *snapshotBuilder) distID(d *claircore.Distribution) (int16, error) {
	if d == nil || (d.DID == "" && d.Name == "" && d.VersionID == "" && d.Version == "" && d.VersionCodeName == "" && d.PrettyName == "" && d.Arch == "" && d.CPE.String() == "") {
		return -1, nil
	}
	key := distKey(*d)
	if id, ok := b.distIx[key]; ok {
		return id, nil
	}
	if len(b.s.dists) > math.MaxInt16 {
		return 0, fmt.Errorf("inmem: distribution dictionary overflow (%d > %d)", len(b.s.dists), math.MaxInt16)
	}
	id := int16(len(b.s.dists))
	b.distIx[key] = id
	b.s.dists = append(b.s.dists, *d)
	return id, nil
}

func (b *snapshotBuilder) finish() (*snapshot, error) {
	s := b.s
	s.nrows = int(b.row)

	display, err := b.display.finish()
	if err != nil {
		return nil, err
	}
	b.display = nil
	s.display = display

	s.cveNames = b.cve.list
	s.pkgNames = b.pkg.list
	s.pkgVers = b.pkgVer.list
	s.modules = b.mod.list
	s.arches = b.arch.list
	s.repoNames = b.rn.list
	s.repoKeys = b.rk.list
	s.repoURIs = b.ru.list
	s.fixedVers = b.fx.list
	s.verKinds = b.vk.list
	s.updaters = b.upd.list

	// Flatten byName into a single contiguous CSR posting slice.
	s.byNamePostings = make([]int32, 0, s.nrows)
	s.byNameSpan = make(map[string]nameSpan, len(b.byName))
	for name, rows := range b.byName {
		start := int32(len(s.byNamePostings))
		s.byNamePostings = append(s.byNamePostings, rows...)
		s.byNameSpan[name] = nameSpan{start: start, count: int32(len(rows))}
	}
	b.byName = nil

	// Persist all primitive column arrays + flat CSR byNamePostings into a
	// read-only mmap file so the Go heap retains only dictionaries and small
	// query-time lookup maps.
	cols, err := writeColsFile(b.dir, s)
	if err != nil {
		display.release()
		return nil, err
	}
	bindCols(s, cols)

	s.knownDists = make([]claircore.Distribution, 0, len(b.knownDists))
	for d := range b.knownDists {
		s.knownDists = append(s.knownDists, d)
	}

	s.moduleIdx = b.mod.m
	s.repoNameIdx = b.rn.m
	s.repoKeyIdx = b.rk.m
	if id, ok := b.fx.lookup(""); ok {
		s.fixedEmptyID = id
	} else {
		s.fixedEmptyID = noMatch
	}

	s.refs.Store(1)
	return s, nil
}

func bindCols(s *snapshot, cols *colsStore) {
	s.cols = cols
	s.cveID = cols.cveID
	s.pkgNameID = cols.pkgNameID
	s.pkgVerID = cols.pkgVerID
	s.kind = cols.kind
	s.moduleID = cols.moduleID
	s.archID = cols.archID
	s.distID = cols.distID
	s.repoNameID = cols.repoNameID
	s.repoKeyID = cols.repoKeyID
	s.repoURIID = cols.repoURIID
	s.fixedID = cols.fixedID
	s.verKindID = cols.verKindID
	s.updaterID = cols.updaterID
	s.normSev = cols.normSev
	s.archop = cols.archop
	s.invert = cols.invert
	s.issued = cols.issued
	s.rangeIdx = cols.rangeIdx
	s.rLower = cols.rLower
	s.rUpper = cols.rUpper
	s.aliasStart = cols.aliasStart
	s.aliasPostings = cols.aliasPostings
	s.selfID = cols.selfID
	s.descI = cols.descI
	s.linkI = cols.linkI
	s.sevI = cols.sevI
	s.byNamePostings = cols.byNamePostings
}

func buildFragment(ctx context.Context, dir, updater string, fragID uint32, maxRows int, it datastore.VulnerabilityIter) (*snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, err := newSnapshotBuilder(dir, fragID, maxRows)
	if err != nil {
		return nil, err
	}
	var iterErr error
	it(func(v *claircore.Vulnerability, err error) bool {
		if ctxErr := ctx.Err(); ctxErr != nil {
			iterErr = ctxErr
			return false
		}
		if err != nil {
			iterErr = err
			return false
		}
		if err := b.add(v); err != nil {
			iterErr = err
			return false
		}
		return true
	})
	if iterErr != nil {
		b.abort()
		return nil, fmt.Errorf("inmem: iterating vulnerabilities for %s: %w", updater, iterErr)
	}
	if err := ctx.Err(); err != nil {
		b.abort()
		return nil, err
	}
	snap, err := b.finish()
	if err != nil {
		b.abort()
		return nil, err
	}
	return snap, nil
}
