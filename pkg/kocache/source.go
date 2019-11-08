package kocache

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/ioutils"
)

func (c *koCache) LoadProbe(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	if c.upstreamBaseURL == "" {
		// Probably offline mode.
		return nil, 0, nil
	}

	entry := c.GetOrAddEntry(filePath)
	if entry == nil {
		return nil, 0, errors.New("kernel object cache is shutting down")
	}
	releaseRef := true
	defer func() {
		if releaseRef {
			entry.ReleaseRef()
		}
	}()

	if !concurrency.WaitInContext(entry.DoneSig(), ctx) {
		return nil, 0, errors.Wrap(ctx.Err(), "context error waiting for download from upstream")
	}

	data, size, err := entry.Contents()
	if err != nil {
		if err == errNotFound {
			err = nil
		}
		return nil, 0, err
	}

	// We need to make sure that `entry` does not get destroyed before reading from the reader is complete, so shift
	// the responsibility to release the reference to the `Close()` method of the returned reader.
	dataReader := io.NewSectionReader(data, 0, size)

	dataReaderWithCloser := ioutils.ReaderWithCloser(dataReader, func() error {
		entry.ReleaseRef()
		return nil
	})
	releaseRef = false // prevent releasing reference upon return
	return dataReaderWithCloser, size, nil
}
