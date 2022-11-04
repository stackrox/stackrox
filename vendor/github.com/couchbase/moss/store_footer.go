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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blevesearch/mmap-go"
)

func (s *Store) persistFooter(file File, footer *Footer,
	options StorePersistOptions) error {
	startTime := time.Now()

	if !options.NoSync {
		err := file.Sync()
		if err != nil {
			return err
		}
	}

	err := s.persistFooterUnsynced(file, footer)
	if err != nil {
		return err
	}

	if !options.NoSync {
		err = file.Sync()
	}

	if err == nil {
		s.histograms["PersistFooterUsecs"].Add(
			uint64(time.Since(startTime).Nanoseconds()/1000), 1)
	}

	return err
}

func (s *Store) persistFooterUnsynced(file File, footer *Footer) error {
	jBuf, err := json.Marshal(footer)
	if err != nil {
		return err
	}

	finfo, err := file.Stat()
	if err != nil {
		return err
	}

	footerPos := pageAlignCeil(finfo.Size())
	footerLen := footerBegLen + len(jBuf) + footerEndLen

	footerBuf := bytes.NewBuffer(make([]byte, 0, footerLen))
	footerBuf.Write(StoreMagicBeg)
	footerBuf.Write(StoreMagicBeg)
	binary.Write(footerBuf, StoreEndian, uint32(StoreVersion))
	binary.Write(footerBuf, StoreEndian, uint32(footerLen))
	footerBuf.Write(jBuf)
	binary.Write(footerBuf, StoreEndian, footerPos)
	binary.Write(footerBuf, StoreEndian, uint32(footerLen))
	footerBuf.Write(StoreMagicEnd)
	footerBuf.Write(StoreMagicEnd)

	footerWritten, err := file.WriteAt(footerBuf.Bytes(), footerPos)
	if err != nil {
		return err
	}
	if footerWritten != len(footerBuf.Bytes()) {
		return fmt.Errorf("store: persistFooter error writing all footerBuf")
	}

	if AllocationGranularity != StorePageSize {
		// Some platforms (windows) only support mmap()'ing at an
		// allocation granularity that's != to a page size.
		//
		// However if on such platforms there are empty segments, then
		// due to the extra space imposed by the above granularity
		// requirement, mmap() can fail complaining about insufficient
		// file space.
		//
		// To avoid this error, simply pad up the file up to a page
		// boundary.  This pad of zeroes will not interfere with file
		// recovery.
		padding := make([]byte, int(pageAlignCeil(int64(footerWritten)))-footerWritten)
		_, err = file.WriteAt(padding, footerPos+int64(footerWritten))
		if err != nil {
			return err
		}
	}

	footer.fileName = finfo.Name()
	footer.filePos = footerPos

	return nil
}

// --------------------------------------------------------

// ReadFooter reads the last valid Footer from a file.
func ReadFooter(options *StoreOptions, file File) (*Footer, error) {
	finfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fref := &FileRef{file: file, refs: 1}

	// To avoid an EOF while reading, start scanning the footer from
	// the last byte. This is under the assumption that the footer is
	// at least 2 bytes long.
	f, err := ScanFooter(options, fref, finfo.Name(), finfo.Size()-1)
	if err != nil {
		return nil, err
	}

	fref.DecRef() // ScanFooter added its own ref-counts on success.

	return f, err
}

// --------------------------------------------------------

// ScanFooter scans a file backwards from the given pos for a valid
// Footer, adding ref-counts to fref on success.
func ScanFooter(options *StoreOptions, fref *FileRef, fileName string,
	pos int64) (*Footer, error) {
	footerBeg := make([]byte, footerBegLen)

	// Align pos to the start of a page (floor).
	pos = pageAlignFloor(pos)

	for {
		for { // Scan for StoreMagicBeg, which may be a potential footer.
			if pos <= 0 {
				return nil, ErrNoValidFooter
			}

			n, err := fref.file.ReadAt(footerBeg, pos)
			if err != nil {
				return nil, err
			}

			if n == footerBegLen &&
				bytes.Equal(StoreMagicBeg, footerBeg[:lenMagicBeg]) &&
				bytes.Equal(StoreMagicBeg, footerBeg[lenMagicBeg:2*lenMagicBeg]) {
				break
			}

			// Move pos back by page size.
			pos -= int64(StorePageSize)
		}

		// Read and check the potential footer.
		footerBegBuf := bytes.NewBuffer(footerBeg[2*lenMagicBeg:])

		var version uint32
		if err := binary.Read(footerBegBuf, StoreEndian, &version); err != nil {
			return nil, err
		}
		if version != StoreVersion {
			return nil, fmt.Errorf("store: version mismatch, "+
				"current: %v != found: %v", StoreVersion, version)
		}

		var length uint32
		if err := binary.Read(footerBegBuf, StoreEndian, &length); err != nil {
			return nil, err
		}

		data := make([]byte, int64(length)-int64(footerBegLen))

		n, err := fref.file.ReadAt(data, pos+int64(footerBegLen))
		if err != nil {
			return nil, err
		}

		if n == len(data) &&
			bytes.Equal(StoreMagicEnd, data[n-lenMagicEnd*2:n-lenMagicEnd]) &&
			bytes.Equal(StoreMagicEnd, data[n-lenMagicEnd:]) {

			content := int(length) - footerBegLen - footerEndLen
			b := bytes.NewBuffer(data[content:])

			var offset int64
			if err = binary.Read(b, StoreEndian, &offset); err != nil {
				return nil, err
			}
			if offset != pos {
				return nil, fmt.Errorf("store: offset mismatch, "+
					"wanted: %v != found: %v", offset, pos)
			}

			var length1 uint32
			if err = binary.Read(b, StoreEndian, &length1); err != nil {
				return nil, err
			}
			if length1 != length {
				return nil, fmt.Errorf("store: length mismatch, "+
					"wanted: %v != found: %v", length1, length)
			}

			f := &Footer{refs: 1, fileName: fileName, filePos: offset}

			err = json.Unmarshal(data[:content], f)
			if err != nil {
				return nil, err
			}

			// json.Unmarshal would have just loaded the map.
			// We now need to load each segment into the map.
			// Also recursively load child footer segment stacks.
			err = f.loadSegments(options, fref)
			if err != nil {
				return nil, err
			}

			return f, nil
		}
		// Else, invalid footer - StoreMagicEnd missing and/or file
		// pos out of bounds.

		// Footer was invalid, so keep scanning.
		pos -= int64(StorePageSize)
	}
}

// --------------------------------------------------------

// loadSegments() loads the segments of a footer.  Adds new ref-counts
// to the fref on success.  The footer will be in an already closed
// state on error.
func (f *Footer) loadSegments(options *StoreOptions, fref *FileRef) (err error) {
	// Track mrefs that we need to DecRef() if there's an error.
	mrefs := make([]*mmapRef, 0, len(f.SegmentLocs))
	mrefs, err = f.doLoadSegments(options, fref, mrefs)
	if err != nil {
		for _, mref := range mrefs {
			mref.DecRef()
		}
		return err
	}

	return nil
}

func (f *Footer) doLoadSegments(options *StoreOptions, fref *FileRef,
	mrefs []*mmapRef) (mrefsSoFar []*mmapRef, err error) {
	// Recursively load the childFooters first.
	for _, childFooter := range f.ChildFooters {
		mrefs, err = childFooter.doLoadSegments(options, fref, mrefs)
		if err != nil {
			return mrefs, err
		}
	}

	if f.ss != nil && f.ss.a != nil {
		return mrefs, nil
	}

	osFile := ToOsFile(fref.file)
	if osFile == nil {
		return mrefs, fmt.Errorf("store: doLoadSegments convert to os.File error")
	}

	a := make([]Segment, len(f.SegmentLocs))

	for i := range f.SegmentLocs {
		sloc := &f.SegmentLocs[i]

		mref := sloc.mref
		if mref != nil {
			if mref.fref != fref {
				return mrefs, fmt.Errorf("store: doLoadSegments fref mismatch")
			}

			mref.AddRef()
			mrefs = append(mrefs, mref)
		} else {
			// We persist kvs before buf, so KvsOffset < BufOffset.
			begOffset := int64(sloc.KvsOffset)
			endOffset := int64(sloc.BufOffset + sloc.BufBytes)

			nbytes := int(endOffset - begOffset)

			// Some platforms (windows) only support mmap()'ing at an
			// allocation granularity that's != to a page size, so
			// calculate the actual offset/nbytes to use.
			begOffsetActual := pageOffset(begOffset, int64(AllocationGranularity))
			begOffsetDelta := int(begOffset - begOffsetActual)
			nbytesActual := nbytes + begOffsetDelta

			// check whether the actual file fits within the footer offsets
			fstats, err := osFile.Stat()
			if err != nil || nbytesActual > int(fstats.Size()) {
				return mrefs, fmt.Errorf("store: doLoadSegments corrupted "+
					"file: %s, err: %+v", fstats.Name(), err)
			}

			mm, err := mmap.MapRegion(osFile, nbytesActual, mmap.RDONLY, 0, begOffsetActual)
			if err != nil {
				return mrefs,
					fmt.Errorf("store: doLoadSegments mmap.Map(),"+
						" begOffsetActual = %v, nbytesActual = %v, sloc = %+v,"+
						" file: %s, file mode: %v, file modification time: %v,"+
						" footer: %+v, f.SegmentLocs: %+v, err: %v",
						begOffsetActual, nbytesActual, sloc,
						fstats.Name(), fstats.Mode(), fstats.ModTime(),
						f, f.SegmentLocs, err)
			}

			fref.AddRef() // New mref owns 1 fref ref-count.

			buf := mm[begOffsetDelta : begOffsetDelta+nbytes]

			sloc.mref = &mmapRef{fref: fref, mm: mm, buf: buf, refs: 1}

			mref = sloc.mref
			mrefs = append(mrefs, mref)

			segmentLoader, exists := SegmentLoaders[sloc.Kind]
			if !exists || segmentLoader == nil {
				return mrefs, fmt.Errorf("store: unknown SegmentLoc kind, sloc: %+v", sloc)
			}

			seg, err := segmentLoader(sloc)
			if err != nil {
				return mrefs, fmt.Errorf("store: segmentLoader failed, footer: %+v,"+
					" f.SegmentLocs: %+v, i: %d, options: %v err: %+v",
					f, f.SegmentLocs, i, options, err)
			}

			segmentKeysIndexMaxBytes := options.SegmentKeysIndexMaxBytes
			if segmentKeysIndexMaxBytes == 0 {
				segmentKeysIndexMaxBytes = DefaultStoreOptions.SegmentKeysIndexMaxBytes
			}

			segmentKeysIndexMinKeyBytes := options.SegmentKeysIndexMinKeyBytes
			if segmentKeysIndexMinKeyBytes == 0 {
				segmentKeysIndexMinKeyBytes = DefaultStoreOptions.SegmentKeysIndexMinKeyBytes
			}

			if segmentKeysIndexMaxBytes > 0 {
				if a, ok := seg.(*segment); ok {
					a.buildIndex(segmentKeysIndexMaxBytes, segmentKeysIndexMinKeyBytes)
				}
			}

			mref.SetExt(seg)
		}

		a[i] = mref.GetExt().(Segment)
	}

	f.ss = &segmentStack{
		options: &options.CollectionOptions,
		a:       a,
		refs:    1,
	}

	return mrefs, nil
}

// --------------------------------------------------------

// ChildCollectionNames returns an array of child collection name strings.
func (f *Footer) ChildCollectionNames() ([]string, error) {
	var childNames = make([]string, len(f.ChildFooters))
	idx := 0
	for name := range f.ChildFooters {
		childNames[idx] = name
		idx++
	}
	return childNames, nil
}

// ChildCollectionSnapshot returns a Snapshot on a given child
// collection by its name.
func (f *Footer) ChildCollectionSnapshot(childCollectionName string) (
	Snapshot, error) {
	childFooter, exists := f.ChildFooters[childCollectionName]
	if !exists {
		return nil, nil
	}
	childFooter.AddRef()
	return childFooter, nil
}

// Close decrements the ref count on this footer
func (f *Footer) Close() error {
	f.DecRef()
	return nil
}

// AddRef increases the ref count on this footer
func (f *Footer) AddRef() {
	f.m.Lock()
	f.refs++
	f.m.Unlock()
}

// DecRef decreases the ref count on this footer
func (f *Footer) DecRef() {
	f.m.Lock()
	f.refs--
	if f.refs <= 0 {
		f.SegmentLocs.DecRef()
		f.SegmentLocs = nil
		f.ss = nil
	}
	f.m.Unlock()
}

// Length returns the length of this footer
func (f *Footer) Length() uint64 {
	jBuf, err := json.Marshal(f)
	if err != nil {
		return 0
	}

	footerLen := footerBegLen + len(jBuf) + footerEndLen
	return uint64(footerLen)
}

// --------------------------------------------------------

// segmentLocs returns the current SegmentLocs and segmentStack for
// a footer, while also incrementing the ref-count on the footer.  The
// caller must DecRef() the footer when done.
func (f *Footer) segmentLocs() (SegmentLocs, *segmentStack) {
	f.m.Lock()

	f.refs++

	slocs, ss := f.SegmentLocs, f.ss

	f.m.Unlock()

	return slocs, ss
}

// --------------------------------------------------------

// Get retrieves a val from the footer, and will return nil val
// if the entry does not exist in the footer.
func (f *Footer) Get(key []byte, readOptions ReadOptions) ([]byte, error) {
	_, ss := f.segmentLocs()
	if ss == nil {
		f.DecRef()
		return nil, nil
	}

	rv, err := ss.Get(key, readOptions)
	if err == nil && rv != nil && !readOptions.NoCopyValue {
		rv = append(make([]byte, 0, len(rv)), rv...) // Copy.
	}

	f.DecRef()

	return rv, err
}

// StartIterator returns a new Iterator instance on this footer.
//
// On success, the returned Iterator will be positioned so that
// Iterator.Current() will either provide the first entry in the
// range or ErrIteratorDone.
//
// A startKeyIncl of nil means the logical "bottom-most" possible key
// and an endKeyExcl of nil means the logical "top-most" possible key.
func (f *Footer) StartIterator(startKeyIncl, endKeyExcl []byte,
	iteratorOptions IteratorOptions) (Iterator, error) {
	_, ss := f.segmentLocs()
	if ss == nil {
		f.DecRef()
		return nil, nil
	}

	iter, err := ss.StartIterator(startKeyIncl, endKeyExcl, iteratorOptions)
	if err != nil || iter == nil {
		f.DecRef()
		return nil, err
	}

	initCloser, ok := iter.(InitCloser)
	if !ok || initCloser == nil {
		iter.Close()
		f.DecRef()
		return nil, ErrUnexpected
	}

	err = initCloser.InitCloser(f)
	if err != nil {
		iter.Close()
		f.DecRef()
		return nil, err
	}

	return iter, nil
}
