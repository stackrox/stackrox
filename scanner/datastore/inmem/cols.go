package inmem

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
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

// writeColsFile serializes all primitive column slices of s into an aligned
// binary file in dir, memory-maps it read-only, and returns a colsStore whose
// slices point directly into the mapped region.
func writeColsFile(dir string, s *snapshot) (*colsStore, error) {
	f, err := os.CreateTemp(dir, colsFilePattern)
	if err != nil {
		return nil, fmt.Errorf("inmem: creating columns file: %w", err)
	}
	path := f.Name()
	cleanupFile := func() {
		if f != nil {
			_ = f.Close()
			f = nil
		}
		if path != "" {
			_ = os.Remove(path)
			path = ""
		}
	}

	bw := bufio.NewWriterSize(f, 256*1024)
	var zeroMeta [colsMetaSize]byte
	if _, err := bw.Write(zeroMeta[:]); err != nil {
		cleanupFile()
		return nil, fmt.Errorf("inmem: writing columns header placeholder: %w", err)
	}

	var (
		secs [colSectionCount]colSecMeta
		off  = uint64(colsMetaSize)
	)
	if err := writeTypedSec(bw, &off, &secs[secCVEID], s.cveID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secPkgNameID], s.pkgNameID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secPkgVerID], s.pkgVerID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secKind], s.kind); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secModuleID], s.moduleID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secArchID], s.archID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secDistID], s.distID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secRepoNameID], s.repoNameID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secRepoKeyID], s.repoKeyID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secRepoURIID], s.repoURIID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secFixedID], s.fixedID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secVerKindID], s.verKindID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secUpdaterID], s.updaterID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secNormSev], s.normSev); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secArchOp], s.archop); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secInvert], s.invert); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secIssued], s.issued); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secRangeIdx], s.rangeIdx); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secRLower], s.rLower); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secRUpper], s.rUpper); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secAliasStart], s.aliasStart); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secAliasPostings], s.aliasPostings); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secSelfID], s.selfID); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secDescI], s.descI); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secLinkI], s.linkI); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secSevI], s.sevI); err != nil {
		cleanupFile()
		return nil, err
	}
	if err := writeTypedSec(bw, &off, &secs[secByNamePostings], s.byNamePostings); err != nil {
		cleanupFile()
		return nil, err
	}

	if err := bw.Flush(); err != nil {
		cleanupFile()
		return nil, fmt.Errorf("inmem: flushing columns file: %w", err)
	}

	var meta [colsMetaSize]byte
	copy(meta[0:8], colsMagic)
	binary.LittleEndian.PutUint32(meta[8:12], colsVersion)
	meta[12] = hostEndian()
	binary.LittleEndian.PutUint32(meta[16:20], uint32(colSectionCount))
	binary.LittleEndian.PutUint32(meta[20:24], uint32(s.nrows))
	for i, sec := range secs {
		base := colsHeaderSize + i*colsSecEntryLen
		binary.LittleEndian.PutUint64(meta[base:base+8], sec.offset)
		binary.LittleEndian.PutUint64(meta[base+8:base+16], sec.count)
	}
	if _, err := f.WriteAt(meta[:], 0); err != nil {
		cleanupFile()
		return nil, fmt.Errorf("inmem: writing columns header: %w", err)
	}

	closeErr := f.Close()
	f = nil
	if closeErr != nil {
		cleanupFile()
		return nil, fmt.Errorf("inmem: closing columns file: %w", closeErr)
	}

	region, err := mapColsRegion(path, int64(off))
	if err != nil {
		cleanupFile()
		return nil, err
	}
	path = ""

	cs, err := parseCols(region, int64(off))
	if err != nil {
		region.release()
		return nil, err
	}
	attachCleanup(region, cs)
	return cs, nil
}

func writeTypedSec[T colElem](w io.Writer, off *uint64, meta *colSecMeta, src []T) error {
	meta.offset = *off
	meta.count = uint64(len(src))
	if len(src) == 0 {
		return nil
	}
	var zero T
	nbytes := uintptr(len(src)) * unsafe.Sizeof(zero)
	//nolint:gosec // G103: zero-copy serialization of pointer-free primitive slice.
	raw := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(src))), nbytes)
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("inmem: writing column section: %w", err)
	}
	*off += uint64(nbytes)
	if rem := *off % colsAlign; rem != 0 {
		pad := colsAlign - rem
		var zeros [colsAlign]byte
		if _, err := w.Write(zeros[:pad]); err != nil {
			return fmt.Errorf("inmem: padding column section: %w", err)
		}
		*off += pad
	}
	return nil
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
	return &mmapRegion{data: data, path: path}, nil
}

// parseCols validates the binary header, endianness, and section bounds of a
// mapped columns file and binds typed read-only slices into region.data.
func parseCols(region *mmapRegion, fileSize int64) (*colsStore, error) {
	data := region.data
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

	var secs [colSectionCount]colSecMeta
	for i := range secs {
		base := colsHeaderSize + i*colsSecEntryLen
		secs[i] = colSecMeta{
			offset: binary.LittleEndian.Uint64(data[base : base+8]),
			count:  binary.LittleEndian.Uint64(data[base+8 : base+16]),
		}
	}

	cs := &colsStore{region: region, fileSize: fileSize}
	var err error
	if cs.cveID, err = readTypedSec[int32](data, secs[secCVEID]); err != nil {
		return nil, err
	}
	if cs.pkgNameID, err = readTypedSec[int32](data, secs[secPkgNameID]); err != nil {
		return nil, err
	}
	if cs.pkgVerID, err = readTypedSec[int32](data, secs[secPkgVerID]); err != nil {
		return nil, err
	}
	if cs.kind, err = readTypedSec[uint8](data, secs[secKind]); err != nil {
		return nil, err
	}
	if cs.moduleID, err = readTypedSec[int32](data, secs[secModuleID]); err != nil {
		return nil, err
	}
	if cs.archID, err = readTypedSec[int32](data, secs[secArchID]); err != nil {
		return nil, err
	}
	if cs.distID, err = readTypedSec[int16](data, secs[secDistID]); err != nil {
		return nil, err
	}
	if cs.repoNameID, err = readTypedSec[int32](data, secs[secRepoNameID]); err != nil {
		return nil, err
	}
	if cs.repoKeyID, err = readTypedSec[int16](data, secs[secRepoKeyID]); err != nil {
		return nil, err
	}
	if cs.repoURIID, err = readTypedSec[int32](data, secs[secRepoURIID]); err != nil {
		return nil, err
	}
	if cs.fixedID, err = readTypedSec[int32](data, secs[secFixedID]); err != nil {
		return nil, err
	}
	if cs.verKindID, err = readTypedSec[int8](data, secs[secVerKindID]); err != nil {
		return nil, err
	}
	if cs.updaterID, err = readTypedSec[int32](data, secs[secUpdaterID]); err != nil {
		return nil, err
	}
	if cs.normSev, err = readTypedSec[uint8](data, secs[secNormSev]); err != nil {
		return nil, err
	}
	if cs.archop, err = readTypedSec[uint8](data, secs[secArchOp]); err != nil {
		return nil, err
	}
	if cs.invert, err = readTypedSec[uint8](data, secs[secInvert]); err != nil {
		return nil, err
	}
	if cs.issued, err = readTypedSec[int64](data, secs[secIssued]); err != nil {
		return nil, err
	}
	if cs.rangeIdx, err = readTypedSec[int32](data, secs[secRangeIdx]); err != nil {
		return nil, err
	}
	if cs.rLower, err = readTypedSec[[10]int32](data, secs[secRLower]); err != nil {
		return nil, err
	}
	if cs.rUpper, err = readTypedSec[[10]int32](data, secs[secRUpper]); err != nil {
		return nil, err
	}
	if cs.aliasStart, err = readTypedSec[int32](data, secs[secAliasStart]); err != nil {
		return nil, err
	}
	if cs.aliasPostings, err = readTypedSec[int32](data, secs[secAliasPostings]); err != nil {
		return nil, err
	}
	if cs.selfID, err = readTypedSec[int32](data, secs[secSelfID]); err != nil {
		return nil, err
	}
	if cs.descI, err = readTypedSec[int32](data, secs[secDescI]); err != nil {
		return nil, err
	}
	if cs.linkI, err = readTypedSec[int32](data, secs[secLinkI]); err != nil {
		return nil, err
	}
	if cs.sevI, err = readTypedSec[int32](data, secs[secSevI]); err != nil {
		return nil, err
	}
	if cs.byNamePostings, err = readTypedSec[int32](data, secs[secByNamePostings]); err != nil {
		return nil, err
	}
	return cs, nil
}

func readTypedSec[T colElem](data []byte, meta colSecMeta) ([]T, error) {
	if meta.count == 0 {
		return nil, nil
	}
	if meta.offset%colsAlign != 0 {
		return nil, fmt.Errorf("inmem: unaligned column section offset %d", meta.offset)
	}
	var zero T
	elemSize := uint64(unsafe.Sizeof(zero))
	if meta.count > (uint64(len(data)))/elemSize {
		return nil, fmt.Errorf("inmem: column section count overflow (%d)", meta.count)
	}
	nbytes := meta.count * elemSize
	if meta.offset > uint64(len(data)) || nbytes > uint64(len(data))-meta.offset {
		return nil, fmt.Errorf("inmem: column section out of bounds (off=%d len=%d file=%d)", meta.offset, nbytes, len(data))
	}
	//nolint:gosec // G103: bounds- and alignment-checked read-only mmap view of pointer-free primitives.
	return unsafe.Slice((*T)(unsafe.Pointer(&data[meta.offset])), meta.count), nil
}
