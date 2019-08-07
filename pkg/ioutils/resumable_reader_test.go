package ioutils

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readerWithErr struct {
	data []byte
	err  error
}

func newReaderWithErr(data string, err error) io.Reader {
	return &readerWithErr{data: []byte(data), err: err}
}

func (r *readerWithErr) Read(buf []byte) (int, error) {
	if len(r.data) == 0 {
		err := r.err
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	toRead := len(buf)
	if toRead > len(r.data) {
		toRead = len(r.data)
	}
	copy(buf[:toRead], r.data[:toRead])
	r.data = r.data[toRead:]
	return toRead, nil
}

func TestResumableReader(t *testing.T) {
	reader, initialAttach, detachmentEvents := NewResumableReader(crc32.NewIEEE())
	defer utils.IgnoreError(reader.Close)

	readResultC := make(chan string)
	go func() {
		var buf [6]byte
		_, err := io.ReadFull(reader, buf[:])
		assert.NoError(t, err)
		assert.NoError(t, reader.Close())

		readResultC <- string(buf[:])
	}()

	errs := []error{
		errors.New("ouch"),
		errors.New("damn"),
		io.EOF,
	}
	readers := []io.Reader{
		newReaderWithErr("fo", errs[0]),
		newReaderWithErr("ob", errs[1]),
		strings.NewReader("ar"),
	}
	partialChecksums := []uint32{
		crc32.ChecksumIEEE([]byte("fo")),
		crc32.ChecksumIEEE([]byte("foob")),
	}

	assert.NoError(t, initialAttach.Attach(readers[0], 0, nil))

	i := 0
	for event := range detachmentEvents {
		assert.Equal(t, readers[i], event.DetachedReader())
		assert.Equal(t, errs[i], event.ReadError())
		assert.Equal(t, int64(2*(i+1)), event.Position())

		if i < 2 {
			var checksum [4]byte
			binary.BigEndian.PutUint32(checksum[:], partialChecksums[i]+3)
			require.Error(t, event.Attach(strings.NewReader("bad checksum"), event.Position(), checksum[:]))
			binary.BigEndian.PutUint32(checksum[:], partialChecksums[i])
			require.Error(t, event.Attach(strings.NewReader("bad pos"), event.Position()+1, checksum[:]))
			require.NoError(t, event.Attach(readers[i+1], event.Position(), checksum[:]))
		} else {
			require.NoError(t, event.Finish(io.EOF))
		}
		i++
	}

	// We can't be sure if there are 2 or 3 iterations - the last read executed by the above `ReadFull` might legally
	// return both 2, io.EOF and 2, nil. In the former case we'd see another detachment event, in the latter we would
	// not.
	assert.True(t, i >= 2 && i <= 3)

	readResult := <-readResultC
	assert.Equal(t, "foobar", readResult)
}
