package inmem

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	colsFilePattern = "inmem-cols-*.col"
	colsMagic       = "ROXCOL01"
	colsVersion     = 1

	colsHeaderSize  = 32
	colsSecEntryLen = 16
	colsAlign       = 8
)

type colSection int

const (
	secCVEID colSection = iota
	secPkgNameID
	secPkgVerID
	secKind
	secModuleID
	secArchID
	secDistID
	secRepoNameID
	secRepoKeyID
	secRepoURIID
	secFixedID
	secVerKindID
	secUpdaterID
	secNormSev
	secArchOp
	secInvert
	secIssued
	secRangeIdx
	secRLower
	secRUpper
	secAliasStart
	secAliasPostings
	secSelfID
	secDescI
	secLinkI
	secSevI
	secByNamePostings

	colSectionCount
)

const colsMetaSize = colsHeaderSize + int(colSectionCount)*colsSecEntryLen

type colElem interface {
	~int8 | ~uint8 | ~int16 | ~int32 | ~int64 | [10]int32
}

type colSecMeta struct {
	offset uint64
	count  uint64
}

// colsStore owns the read-only memory-mapped binary column file backing a
// fragment's primitive per-row arrays and flat CSR byName postings.
type colsStore struct {
	region   *mmapRegion
	fileSize int64
	nrows    int

	cveID          []int32
	pkgNameID      []int32
	pkgVerID       []int32
	kind           []uint8
	moduleID       []int32
	archID         []int32
	distID         []int16
	repoNameID     []int32
	repoKeyID      []int16
	repoURIID      []int32
	fixedID        []int32
	verKindID      []int8
	updaterID      []int32
	normSev        []uint8
	archop         []uint8
	invert         []uint8
	issued         []int64
	rangeIdx       []int32
	rLower         [][10]int32
	rUpper         [][10]int32
	aliasStart     []int32
	aliasPostings  []int32
	selfID         []int32
	descI          []int32
	linkI          []int32
	sevI           []int32
	byNamePostings []int32
}

func (c *colsStore) offHeapBytes() int64 {
	if c == nil {
		return 0
	}
	return c.fileSize
}

func (c *colsStore) release() {
	if c != nil && c.region != nil {
		c.region.release()
		runtime.KeepAlive(c)
	}
}

// hostEndian returns 0 on little-endian hosts and 1 on big-endian hosts.
func hostEndian() uint8 {
	var v uint16 = 1
	//nolint:gosec // G103: standard single-byte endianness probe.
	if *(*byte)(unsafe.Pointer(&v)) == 1 {
		return 0
	}
	return 1
}

type colWriter struct {
	w    io.Writer
	off  uint64
	secs [colSectionCount]colSecMeta
	err  error
}

func writeSec[T colElem](cw *colWriter, sec colSection, src []T) {
	if cw.err != nil {
		return
	}
	cw.secs[sec] = colSecMeta{offset: cw.off, count: uint64(len(src))}
	if len(src) == 0 {
		return
	}
	var zero T
	nbytes := uintptr(len(src)) * unsafe.Sizeof(zero)
	//nolint:gosec // G103: zero-copy serialization of pointer-free primitive slice.
	raw := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(src))), nbytes)
	if _, err := cw.w.Write(raw); err != nil {
		cw.err = fmt.Errorf("inmem: writing column section %d: %w", sec, err)
		return
	}
	cw.off += uint64(nbytes)
	if rem := cw.off % colsAlign; rem != 0 {
		pad := colsAlign - rem
		var zeros [colsAlign]byte
		if _, err := cw.w.Write(zeros[:pad]); err != nil {
			cw.err = fmt.Errorf("inmem: padding column section %d: %w", sec, err)
			return
		}
		cw.off += pad
	}
}

// writeColsFile serializes all primitive column slices of s into an aligned
// binary file in dir, memory-maps it read-only, and returns a colsStore whose
// slices point directly into the mapped region.
func writeColsFile(dir string, s *snapshot) (*colsStore, error) {
	if s.nrows < 0 || s.nrows > math.MaxInt32 {
		return nil, fmt.Errorf("inmem: invalid row count %d", s.nrows)
	}
	f, err := os.CreateTemp(dir, colsFilePattern)
	if err != nil {
		return nil, fmt.Errorf("inmem: creating columns file: %w", err)
	}
	path := f.Name()
	var ok bool
	defer func() {
		if ok {
			return
		}
		if f != nil {
			_ = f.Close()
		}
		if path != "" {
			_ = os.Remove(path)
		}
	}()

	bw := bufio.NewWriterSize(f, blobWriteBufSize)
	var zeroMeta [colsMetaSize]byte
	if _, err := bw.Write(zeroMeta[:]); err != nil {
		return nil, fmt.Errorf("inmem: writing columns header placeholder: %w", err)
	}

	cw := &colWriter{w: bw, off: uint64(colsMetaSize)}
	writeSec(cw, secCVEID, s.cveID)
	writeSec(cw, secPkgNameID, s.pkgNameID)
	writeSec(cw, secPkgVerID, s.pkgVerID)
	writeSec(cw, secKind, s.kind)
	writeSec(cw, secModuleID, s.moduleID)
	writeSec(cw, secArchID, s.archID)
	writeSec(cw, secDistID, s.distID)
	writeSec(cw, secRepoNameID, s.repoNameID)
	writeSec(cw, secRepoKeyID, s.repoKeyID)
	writeSec(cw, secRepoURIID, s.repoURIID)
	writeSec(cw, secFixedID, s.fixedID)
	writeSec(cw, secVerKindID, s.verKindID)
	writeSec(cw, secUpdaterID, s.updaterID)
	writeSec(cw, secNormSev, s.normSev)
	writeSec(cw, secArchOp, s.archop)
	writeSec(cw, secInvert, s.invert)
	writeSec(cw, secIssued, s.issued)
	writeSec(cw, secRangeIdx, s.rangeIdx)
	writeSec(cw, secRLower, s.rLower)
	writeSec(cw, secRUpper, s.rUpper)
	writeSec(cw, secAliasStart, s.aliasStart)
	writeSec(cw, secAliasPostings, s.aliasPostings)
	writeSec(cw, secSelfID, s.selfID)
	writeSec(cw, secDescI, s.descI)
	writeSec(cw, secLinkI, s.linkI)
	writeSec(cw, secSevI, s.sevI)
	writeSec(cw, secByNamePostings, s.byNamePostings)
	if cw.err != nil {
		return nil, cw.err
	}
	if err := bw.Flush(); err != nil {
		return nil, fmt.Errorf("inmem: flushing columns file: %w", err)
	}

	var meta [colsMetaSize]byte
	copy(meta[0:8], colsMagic)
	binary.LittleEndian.PutUint32(meta[8:12], colsVersion)
	meta[12] = hostEndian()
	binary.LittleEndian.PutUint32(meta[16:20], uint32(colSectionCount))
	binary.LittleEndian.PutUint32(meta[20:24], uint32(s.nrows))
	for i, sec := range cw.secs {
		base := colsHeaderSize + i*colsSecEntryLen
		binary.LittleEndian.PutUint64(meta[base:base+8], sec.offset)
		binary.LittleEndian.PutUint64(meta[base+8:base+16], sec.count)
	}
	if _, err := f.WriteAt(meta[:], 0); err != nil {
		return nil, fmt.Errorf("inmem: writing columns header: %w", err)
	}

	closeErr := f.Close()
	f = nil
	if closeErr != nil {
		return nil, fmt.Errorf("inmem: closing columns file: %w", closeErr)
	}

	region, err := mapColsRegion(path, int64(cw.off))
	if err != nil {
		return nil, err
	}
	path = "" // ownership transferred to region

	cs, err := parseCols(region, int64(cw.off))
	if err != nil {
		region.release()
		return nil, err
	}
	attachCleanup(region, cs)
	ok = true
	return cs, nil
}

func mapColsRegion(path string, size int64) (*mmapRegion, error) {
	if size <= 0 {
		_ = os.Remove(path)
		return nil, errors.New("inmem: empty columns file")
	}
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("inmem: opening columns file for mmap: %w", err)
	}
	data, mmapErr := unix.Mmap(int(fd.Fd()), 0, int(size), unix.PROT_READ, unix.MAP_SHARED)
	closeErr := fd.Close()
	if mmapErr != nil {
		return nil, fmt.Errorf("inmem: mmap columns file: %w", mmapErr)
	}
	if closeErr != nil {
		_ = unix.Munmap(data)
		return nil, fmt.Errorf("inmem: closing columns fd: %w", closeErr)
	}
	return &mmapRegion{handle: &mmapHandle{data: data, path: path}}, nil
}

type colReader struct {
	data []byte
	secs [colSectionCount]colSecMeta
	err  error
}

func readSec[T colElem](cr *colReader, sec colSection) []T {
	if cr.err != nil {
		return nil
	}
	meta := cr.secs[sec]
	if meta.count == 0 {
		return nil
	}
	if meta.offset%colsAlign != 0 {
		cr.err = fmt.Errorf("inmem: unaligned column section %d offset %d", sec, meta.offset)
		return nil
	}
	var zero T
	elemSize := uint64(unsafe.Sizeof(zero))
	if meta.count > uint64(len(cr.data))/elemSize {
		cr.err = fmt.Errorf("inmem: column section %d count overflow (%d)", sec, meta.count)
		return nil
	}
	nbytes := meta.count * elemSize
	if meta.offset > uint64(len(cr.data)) || nbytes > uint64(len(cr.data))-meta.offset {
		cr.err = fmt.Errorf("inmem: column section %d out of bounds (off=%d len=%d file=%d)", sec, meta.offset, nbytes, len(cr.data))
		return nil
	}
	//nolint:gosec // G103: bounds- and alignment-checked read-only mmap view of pointer-free primitives.
	return unsafe.Slice((*T)(unsafe.Pointer(&cr.data[meta.offset])), meta.count)
}

// parseCols validates the binary header, endianness, section bounds, and
// cross-section row counts of a mapped columns file and binds typed read-only
// slices into region.data().
func parseCols(region *mmapRegion, fileSize int64) (*colsStore, error) {
	data := region.data()
	if len(data) < colsMetaSize {
		return nil, fmt.Errorf("inmem: columns file too short (%d < %d)", len(data), colsMetaSize)
	}
	if string(data[0:8]) != colsMagic {
		return nil, fmt.Errorf("inmem: invalid columns magic %q", string(data[0:8]))
	}
	if ver := binary.LittleEndian.Uint32(data[8:12]); ver != colsVersion {
		return nil, fmt.Errorf("inmem: unsupported columns version %d (want %d)", ver, colsVersion)
	}
	if endian := data[12]; endian != hostEndian() {
		return nil, fmt.Errorf("inmem: columns endianness mismatch (file=%d, host=%d)", endian, hostEndian())
	}
	if nsec := binary.LittleEndian.Uint32(data[16:20]); nsec != uint32(colSectionCount) {
		return nil, fmt.Errorf("inmem: unexpected columns section count %d (want %d)", nsec, colSectionCount)
	}
	nrows := int(binary.LittleEndian.Uint32(data[20:24]))

	cr := &colReader{data: data}
	for i := range cr.secs {
		base := colsHeaderSize + i*colsSecEntryLen
		cr.secs[i] = colSecMeta{
			offset: binary.LittleEndian.Uint64(data[base : base+8]),
			count:  binary.LittleEndian.Uint64(data[base+8 : base+16]),
		}
	}

	cs := &colsStore{
		region:         region,
		fileSize:       fileSize,
		nrows:          nrows,
		cveID:          readSec[int32](cr, secCVEID),
		pkgNameID:      readSec[int32](cr, secPkgNameID),
		pkgVerID:       readSec[int32](cr, secPkgVerID),
		kind:           readSec[uint8](cr, secKind),
		moduleID:       readSec[int32](cr, secModuleID),
		archID:         readSec[int32](cr, secArchID),
		distID:         readSec[int16](cr, secDistID),
		repoNameID:     readSec[int32](cr, secRepoNameID),
		repoKeyID:      readSec[int16](cr, secRepoKeyID),
		repoURIID:      readSec[int32](cr, secRepoURIID),
		fixedID:        readSec[int32](cr, secFixedID),
		verKindID:      readSec[int8](cr, secVerKindID),
		updaterID:      readSec[int32](cr, secUpdaterID),
		normSev:        readSec[uint8](cr, secNormSev),
		archop:         readSec[uint8](cr, secArchOp),
		invert:         readSec[uint8](cr, secInvert),
		issued:         readSec[int64](cr, secIssued),
		rangeIdx:       readSec[int32](cr, secRangeIdx),
		rLower:         readSec[[10]int32](cr, secRLower),
		rUpper:         readSec[[10]int32](cr, secRUpper),
		aliasStart:     readSec[int32](cr, secAliasStart),
		aliasPostings:  readSec[int32](cr, secAliasPostings),
		selfID:         readSec[int32](cr, secSelfID),
		descI:          readSec[int32](cr, secDescI),
		linkI:          readSec[int32](cr, secLinkI),
		sevI:           readSec[int32](cr, secSevI),
		byNamePostings: readSec[int32](cr, secByNamePostings),
	}
	if cr.err != nil {
		return nil, cr.err
	}
	if err := validateColCounts(cs, nrows); err != nil {
		return nil, err
	}
	return cs, nil
}

func validateColCounts(cs *colsStore, nrows int) error {
	perRow := []struct {
		name string
		n    int
	}{
		{"cveID", len(cs.cveID)},
		{"pkgNameID", len(cs.pkgNameID)},
		{"pkgVerID", len(cs.pkgVerID)},
		{"kind", len(cs.kind)},
		{"moduleID", len(cs.moduleID)},
		{"archID", len(cs.archID)},
		{"distID", len(cs.distID)},
		{"repoNameID", len(cs.repoNameID)},
		{"repoKeyID", len(cs.repoKeyID)},
		{"repoURIID", len(cs.repoURIID)},
		{"fixedID", len(cs.fixedID)},
		{"verKindID", len(cs.verKindID)},
		{"updaterID", len(cs.updaterID)},
		{"normSev", len(cs.normSev)},
		{"archop", len(cs.archop)},
		{"invert", len(cs.invert)},
		{"issued", len(cs.issued)},
		{"rangeIdx", len(cs.rangeIdx)},
		{"selfID", len(cs.selfID)},
		{"descI", len(cs.descI)},
		{"linkI", len(cs.linkI)},
		{"sevI", len(cs.sevI)},
		{"byNamePostings", len(cs.byNamePostings)},
	}
	for _, col := range perRow {
		if col.n != nrows {
			return fmt.Errorf("inmem: column %s length %d != nrows %d", col.name, col.n, nrows)
		}
	}
	if len(cs.aliasStart) != nrows+1 {
		return fmt.Errorf("inmem: aliasStart length %d != nrows+1 (%d)", len(cs.aliasStart), nrows+1)
	}
	if len(cs.rLower) != len(cs.rUpper) {
		return fmt.Errorf("inmem: range lower length %d != upper length %d", len(cs.rLower), len(cs.rUpper))
	}
	return nil
}
