//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/ghistogram"
)

// TODO: Improved version parsers / checkers / handling (semver?).

// StorePrefix is the file name prefix
var StorePrefix = "data-"

// StoreSuffix is the file name suffix
var StoreSuffix = ".moss"

// StoreEndian is the preferred endianness used by moss
var StoreEndian = binary.LittleEndian

// StorePageSize is the page size used by moss
var StorePageSize = 4096

// StoreVersion must be bumped whenever the file format changes.
var StoreVersion = uint32(4)

// StoreMagicBeg is the magic byte sequence at the start of a footer
var StoreMagicBeg = []byte("0m1o2s")

// StoreMagicEnd is the magic byte sequence at the end of a footer
var StoreMagicEnd = []byte("3s4p5s")

var lenMagicBeg = len(StoreMagicBeg)
var lenMagicEnd = len(StoreMagicEnd)

// footerBegLen includes StoreVersion(uint32) & footerLen(uint32).
var footerBegLen = lenMagicBeg + lenMagicBeg + 4 + 4

// footerEndLen includes footerOffset(int64) & footerLen(uint32) again.
var footerEndLen = 8 + 4 + lenMagicEnd + lenMagicEnd

// --------------------------------------------------------

// Header represents the JSON stored at the head of a file, where the
// file header bytes should be less than StorePageSize length.
type Header struct {
	Version       uint32 // The file format / StoreVersion.
	CreatedAt     string
	CreatedEndian string // The endian() of the file creator.
}

// Footer represents a footer record persisted in a file, and also
// implements the moss.Snapshot interface.
type Footer struct {
	m    sync.Mutex // Protects the fields that follow.
	refs int

	SegmentLocs      SegmentLocs // Persisted; older SegmentLoc's come first.
	PrevFooterOffset int64       // Persisted; link for snapshot restoration.

	ss *segmentStack // Ephemeral.

	fileName string // Ephemeral; file name; "" when unpersisted.
	filePos  int64  // Ephemeral; byte offset of footer; <= 0 when unpersisted.

	incarNum uint64 // Ephemeral; to detect fast collection recreations.

	ChildFooters map[string]*Footer // Persisted; Child collections by name.
}

// --------------------------------------------------------

// persist helps the store implement the lower-level-update func.  The
// higher snapshot may be nil.
func (s *Store) persist(higher Snapshot, persistOptions StorePersistOptions) (
	Snapshot, error) {
	wasCompacted, err := s.compactMaybe(higher, persistOptions)
	if err != nil {
		return nil, err
	}
	if wasCompacted {
		return s.Snapshot()
	}

	// If no dirty higher items, we're still clean, so just snapshot.
	// If in case of ReadOnly mode, just snapshot.
	if higher == nil || s.Options().CollectionOptions.ReadOnly {
		return s.Snapshot()
	}

	startTime := time.Now()

	ss, ok := higher.(*segmentStack)
	if !ok {
		return nil, fmt.Errorf("store: can only persist segmentStack")
	}

	// If higher segment has no data, we're still clean, so just snapshot.
	if ss.isEmpty() {
		return s.Snapshot()
	}

	fref, file, err := s.startOrReuseFile()
	if err != nil {
		return nil, err
	}
	defer fref.DecRef()

	// TODO: Pre-allocate file space up front?

	// Recursively sort all child collection stacks if sorting was deferred.
	ss.ensureFullySorted()

	// Recursively build a new store footer combined with higher snapshot.
	s.m.Lock()
	footer := s.buildNewFooter(s.footer, ss)
	s.m.Unlock()

	// Recursively write out all the segments of the snapshot.
	err = s.persistSegments(ss, footer, file, fref)
	if err != nil {
		return nil, err
	}

	// Recursively load all segments of the newly persisted footer.
	err = footer.loadSegments(s.options, fref)
	if err != nil {
		return nil, err
	}

	// Recursively persist all footers of top-level and child collections.
	err = s.persistFooter(file, footer, persistOptions)
	if err != nil {
		footer.DecRef()
		return nil, err
	}

	footer.AddRef() // One ref-count will be held by the store.

	s.m.Lock()
	prevFooter := s.footer
	s.footer = footer
	s.totPersists++
	s.m.Unlock()

	s.histograms["PersistUsecs"].Add(
		uint64(time.Since(startTime).Nanoseconds()/1000), 1)

	if prevFooter != nil {
		prevFooter.DecRef()
	}

	return footer, nil // The other ref-count returned to caller.
}

// buildNewFooter will construct a new Footer for the store by combining
// the given storeFooter's segmentLocs with that of the incoming snapshot.
func (s *Store) buildNewFooter(storeFooter *Footer, ss *segmentStack) *Footer {
	footer := &Footer{refs: 1, incarNum: ss.incarNum}

	numSegmentLocs := len(ss.a)
	var segmentLocs []SegmentLoc
	if storeFooter != nil {
		numSegmentLocs += len(storeFooter.SegmentLocs)
		segmentLocs = make([]SegmentLoc, 0, numSegmentLocs)
		segmentLocs = append(segmentLocs, storeFooter.SegmentLocs...)
		footer.PrevFooterOffset = storeFooter.filePos
	} else {
		segmentLocs = make([]SegmentLoc, 0, numSegmentLocs)
	}
	footer.SegmentLocs = segmentLocs

	// Now process the child collections recursively.
	for cName, childStack := range ss.childSegStacks {
		var storeChildFooter *Footer
		if storeFooter != nil && storeFooter.ChildFooters != nil {
			var exists bool
			storeChildFooter, exists = storeFooter.ChildFooters[cName]
			if exists {
				if storeChildFooter.incarNum != childStack.incarNum {
					// This is a special case of deletion & recreate where an
					// existing child collection has been deleted and quickly
					// recreated. Here we drop the existing store footer's
					// segments that correspond to the prior incarnation.
					storeChildFooter = nil
				}
			}
		}

		childFooter := s.buildNewFooter(storeChildFooter, childStack)
		if len(footer.ChildFooters) == 0 {
			footer.ChildFooters = make(map[string]*Footer)
		}
		footer.ChildFooters[cName] = childFooter
	}

	// As a deleted Child collection does not feature in the source
	// segmentStack, its corresponding Footer would simply get dropped.
	return footer
}

// persistSegments will recursively write out all the segments of the
// current collection as well as any of its child collections.
func (s *Store) persistSegments(ss *segmentStack, footer *Footer,
	file File, fref *FileRef) error {
	// First persist the child segments recursively.
	for cName, childSegStack := range ss.childSegStacks {
		err := s.persistSegments(childSegStack, footer.ChildFooters[cName],
			file, fref)
		if err != nil {
			return err
		}
	}

	for _, segment := range ss.a {
		if segment.Len() <= 0 {
			// With multiple child collections it is possible that some child
			// collections segments are empty. Ok to skip these empty segments.
			continue
		}

		segmentLoc, err := s.persistSegment(file, segment, s.options)
		if err != nil {
			return err
		}

		footer.SegmentLocs = append(footer.SegmentLocs, segmentLoc)
	}

	return nil
}

// --------------------------------------------------------

// startOrReuseFile either creates a new file or reuses the file from
// the last/current footer.
func (s *Store) startOrReuseFile() (fref *FileRef, file File, err error) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.footer != nil {
		slocs, _ := s.footer.segmentLocs()
		defer s.footer.DecRef()

		if len(slocs) > 0 {
			fref := slocs[0].mref.fref
			file := fref.AddRef()

			return fref, file, nil
		}
	}

	return s.startFileLOCKED()
}

func (s *Store) startFileLOCKED() (*FileRef, File, error) {
	fname, file, err := s.createNextFileLOCKED()
	if err != nil {
		return nil, nil, err
	}

	if err = s.persistHeader(file); err != nil {
		file.Close()

		os.Remove(path.Join(s.dir, fname))

		return nil, nil, err
	}

	fref := &FileRef{file: file, refs: 1}
	s.fileRefMap[fname] = fref

	return fref, file, nil
}

func (s *Store) createNextFileLOCKED() (string, File, error) {
	// File to be opened in RDWR mode here because this is either
	// invoked by the persister or the compactor either of which
	// do not execute in the ReadOnly mode
	fname := FormatFName(s.nextFNameSeq)
	s.nextFNameSeq++

	file, err := s.options.OpenFile(path.Join(s.dir, fname),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", nil, err
	}

	return fname, file, nil
}

// removeFileOnClose will setup the callback to wipe out the file safely
// when all references to it are closed.
func (s *Store) removeFileOnClose(fref *FileRef) (os.FileInfo, error) {
	finfo, err := fref.file.Stat()
	if err != nil {
		return nil, err
	}

	if len(finfo.Name()) > 0 {
		fref.OnAfterClose(func() {
			fileName := finfo.Name()
			go func() {
				s.m.Lock()
				delete(s.fileRefMap, fileName)
				s.m.Unlock()
				err := os.Remove(path.Join(s.dir, fileName))
				if err != nil {
					if s.options.CollectionOptions.Log != nil {
						s.options.CollectionOptions.Log(
							"store: error deleting file %s, err: %v",
							fileName, err)
					}
				}
			}()
		})
	}

	return finfo, nil
}

// --------------------------------------------------------

// Fetch all the files within the store, and the number of those
// files that are open/in-use.
func (s *Store) allFiles() (map[string]interface{}, int) {
	files := make(map[string]interface{})

	s.m.Lock()
	for filename, ref := range s.fileRefMap {
		if ref != nil {
			files[filename] =
				map[string]interface{}{"ref_count": ref.FetchRefCount()}
		}
	}
	s.m.Unlock()

	numFilesOpen := len(files)

	fd, err := os.Open(s.dir)
	if err == nil {
		filelist, err := fd.Readdir(-1)
		fd.Close()
		if err == nil {
			for _, finfo := range filelist {
				fileEntry := map[string]interface{}{"ref_count": nil,
					"file_size":     finfo.Size(),
					"file_mode":     finfo.Mode(),
					"file_modified": finfo.ModTime(),
					"is_dir":        finfo.IsDir()}

				referencedEntry, exists := files[finfo.Name()]
				if exists {
					refMap, _ := referencedEntry.(map[string]interface{})
					fileEntry["ref_count"] = refMap["ref_count"]
				}

				files[finfo.Name()] = fileEntry
			}
		}
	}

	return files, numFilesOpen
}

// --------------------------------------------------------

// HeaderLength returns the length of the header
func HeaderLength() uint64 {
	return uint64(StorePageSize)
}

func (s *Store) persistHeader(file File) error {
	buf, err := json.Marshal(Header{
		Version:       StoreVersion,
		CreatedAt:     time.Now().Format(time.RFC3339),
		CreatedEndian: endian(),
	})
	if err != nil {
		return err
	}

	str := "moss-data-store:\n" + string(buf) + "\n"
	if len(str) >= StorePageSize {
		return fmt.Errorf("store: header size too big")
	}
	str = str + strings.Repeat("\n", StorePageSize-len(str))

	n, err := file.WriteAt([]byte(str), 0)
	if err != nil {
		return err
	}
	if n != len(str) {
		return fmt.Errorf("store: could not write full header")
	}

	return nil
}

func checkHeader(file File) error {
	buf := make([]byte, StorePageSize)

	n, err := file.ReadAt(buf, int64(0))
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("store: readHeader too short")
	}

	lines := strings.Split(string(buf), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("store: readHeader not enough lines")
	}
	if lines[0] != "moss-data-store:" {
		return fmt.Errorf("store: readHeader wrong file prefix")
	}

	hdr := Header{}
	err = json.Unmarshal([]byte(lines[1]), &hdr)
	if err != nil {
		return err
	}
	if hdr.Version != StoreVersion {
		return fmt.Errorf("store: readHeader wrong version")
	}
	if hdr.CreatedEndian != endian() {
		return fmt.Errorf("store: readHeader endian of file was: %s, need: %s",
			hdr.CreatedEndian, endian())
	}

	return nil
}

// --------------------------------------------------------

func (s *Store) persistSegment(file File, segIn Segment,
	options *StoreOptions) (rv SegmentLoc, err error) {
	segPersister, ok := segIn.(SegmentPersister)
	if !ok {
		return rv, fmt.Errorf("store: can only persist SegmentPersister type")
	}
	if s.IsAborted() {
		return rv, ErrAborted
	}
	return segPersister.Persist(file, options)
}

// --------------------------------------------------------

// ParseFNameSeq parses a file name like "data-000123.moss" into 123.
func ParseFNameSeq(fname string) (int64, error) {
	if len(StorePrefix) > len(fname)-len(StoreSuffix) {
		return 0, fmt.Errorf("store: invalid filename: %s, filename too short", fname)
	}
	seqStr := fname[len(StorePrefix) : len(fname)-len(StoreSuffix)]
	return strconv.ParseInt(seqStr, 16, 64)
}

// FormatFName returns a file name like "data-000123.moss" given a seq of 123.
func FormatFName(seq int64) string {
	return fmt.Sprintf("%s%016x%s", StorePrefix, seq, StoreSuffix)
}

// --------------------------------------------------------

// pageAlignCeil returns the pos if it's at the start of a page.
// Else, pageAlignCeil() returns pos bumped up to the next multiple
// of StorePageSize.
func pageAlignCeil(pos int64) int64 {
	rem := pos % int64(StorePageSize)
	if rem != 0 {
		return pos + int64(StorePageSize) - rem
	}
	return pos
}

// pageAlignFloor returns the pos if it's at the start of a page.
// Else, pageAlignFloor() returns pos bumped down to the previous
// multiple of StorePageSize.
func pageAlignFloor(pos int64) int64 {
	rem := pos % int64(StorePageSize)
	if rem != 0 {
		return pos - rem
	}
	return pos
}

// pageOffset returns the page offset for a given pos.
func pageOffset(pos, pageSize int64) int64 {
	rem := pos % pageSize
	if rem != 0 {
		return pos - rem
	}
	return pos
}

// --------------------------------------------------------

func openStore(dir string, options StoreOptions) (*Store, error) {
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var maxFNameSeq int64

	var fnames []string
	for _, fileInfo := range fileInfos { // Find candidate file names.
		fname := fileInfo.Name()
		if strings.HasPrefix(fname, StorePrefix) &&
			strings.HasSuffix(fname, StoreSuffix) {
			fnames = append(fnames, fname)
		}

		fnameSeq, err := ParseFNameSeq(fname)
		if err == nil && fnameSeq > maxFNameSeq {
			maxFNameSeq = fnameSeq
		}
	}

	if options.OpenFile == nil {
		options.OpenFile =
			func(name string, flag int, perm os.FileMode) (File, error) {
				return os.OpenFile(name, flag, perm)
			}
	}

	histograms := make(ghistogram.Histograms)
	histograms["PersistFooterUsecs"] =
		ghistogram.NewNamedHistogram("PersistFooterUsecs", 10, 4, 4)
	histograms["PersistUsecs"] =
		ghistogram.NewNamedHistogram("PersistUsecs", 10, 4, 4)
	histograms["CompactUsecs"] =
		ghistogram.NewNamedHistogram("CompactUsecs", 10, 4, 4)

	if len(fnames) <= 0 {
		emptyFooter := &Footer{
			refs:         1,
			ss:           &segmentStack{options: &options.CollectionOptions},
			ChildFooters: make(map[string]*Footer),
		}

		return &Store{
			dir:          dir,
			options:      &options,
			refs:         1,
			footer:       emptyFooter,
			nextFNameSeq: 1,
			histograms:   histograms,
			fileRefMap:   make(map[string]*FileRef),
			abortCh:      make(chan struct{}),
		}, nil
	}

	sort.Strings(fnames)
	if options.CollectionOptions.Log != nil {
		options.CollectionOptions.Log("store: openStore,"+
			" files found: %q", fnames)
	}

	for i := len(fnames) - 1; i >= 0; i-- {
		var flag int
		var perm os.FileMode
		if options.CollectionOptions.ReadOnly {
			flag = os.O_RDONLY
			perm = 0400
		} else {
			flag = os.O_RDWR
			perm = 0600
		}

		file, err := options.OpenFile(path.Join(dir, fnames[i]), flag, perm)
		if err != nil {
			continue
		}

		err = checkHeader(file)
		if err != nil {
			file.Close()
			return nil, err
		}

		// Will recursively restore ChildFooters of childCollections
		footer, err := ReadFooter(&options, file) // Footer owns file on success.
		if err != nil {
			file.Close()
			continue
		}

		if !options.KeepFiles {
			rmFiles := append(fnames[0:i], fnames[i+1:]...)
			if options.CollectionOptions.Log != nil {
				options.CollectionOptions.Log("store: openStore,"+
					" files to remove: %q", rmFiles)
			}

			err := removeFiles(dir, rmFiles)
			if err != nil {
				footer.Close()
				return nil, err
			}
		}

		return &Store{
			dir:          dir,
			options:      &options,
			refs:         1,
			footer:       footer,
			nextFNameSeq: maxFNameSeq + 1,
			histograms:   histograms,
			fileRefMap:   make(map[string]*FileRef),
			abortCh:      make(chan struct{}),
		}, nil
	}

	return nil, fmt.Errorf("store: could not open/parse"+
		" any file, dir: %s", dir)
}

// --------------------------------------------------------

func (s *Store) openCollection(
	options StoreOptions,
	persistOptions StorePersistOptions) (Collection, error) {
	storeSnapshotInit, err := s.Snapshot()
	if err != nil {
		return nil, err
	}

	co := options.CollectionOptions
	co.LowerLevelInit = storeSnapshotInit
	co.LowerLevelUpdate = func(higher Snapshot) (Snapshot, error) {
		var ss Snapshot
		var erro error
		ss, erro = s.Persist(higher, persistOptions)
		if erro != nil {
			return nil, erro
		}

		if storeSnapshotInit != nil {
			storeSnapshotInit.Close()
			storeSnapshotInit = nil
		}

		return ss, erro
	}

	storeFooter, ok := storeSnapshotInit.(*Footer)
	if !ok {
		storeSnapshotInit.Close()
		return nil, fmt.Errorf("store: wrong storeSnapshotInit type")
	}

	coll, err := restoreCollection(&co, storeFooter)
	if err != nil {
		storeSnapshotInit.Close()
		return nil, err
	}

	err = coll.Start()
	if err != nil {
		storeSnapshotInit.Close()
		return nil, err
	}

	return coll, nil
}

// statsReporter interface represents stats reporting methods.
type statsReporter interface {
	reportBytesWritten(numBytesWritten uint64)
}

func (s *Store) reportBytesWritten(numBytesWritten uint64) {
	s.m.Lock()
	s.totCompactionWrittenBytes += numBytesWritten
	s.m.Unlock()
}

func restoreCollection(co *CollectionOptions, storeFooter *Footer) (
	rv *collection, err error) {
	var coll *collection
	if storeFooter.incarNum == 0 {
		var newColl Collection
		newColl, err = NewCollection(*co)
		if err != nil {
			return nil, err
		}

		var ok bool
		coll, ok = newColl.(*collection)
		if !ok {
			return nil, fmt.Errorf("store: wrong collection type")
		}
	} else {
		coll = &collection{
			options:  co,
			stats:    &CollectionStats{},
			incarNum: storeFooter.incarNum,
		}
	}

	coll.highestIncarNum = coll.incarNum

	for collName, childFooter := range storeFooter.ChildFooters {
		if len(coll.childCollections) == 0 {
			coll.childCollections = make(map[string]*collection)
		}

		// Keep the incarnation numbers of the newly restored child
		// collections monotonically increasing.
		coll.highestIncarNum++
		childFooter.incarNum = coll.highestIncarNum

		var childCollection *collection
		childCollection, err = restoreCollection(co, childFooter)
		if err != nil {
			break
		}

		coll.childCollections[collName] = childCollection
	}

	return coll, err
}

// --------------------------------------------------------

func removeFiles(dir string, fnames []string) error {
	for _, fname := range fnames {
		err := os.Remove(path.Join(dir, fname))
		if err != nil {
			return err
		}
	}

	return nil
}
