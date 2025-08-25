package handler

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	blobDatastoreMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOfflineFileManager(t *testing.T) {
	ctx := context.Background()
	blobName := "fake"

	t.Run("panic in non-release builds if blob already registered", func(t *testing.T) {
		fm := newOfflineFileManager(nil, "")
		fm.Register(blobName)
		registerAgain := func() { fm.Register(blobName) }

		if !buildinfo.ReleaseBuild {
			require.Panics(t, registerAgain)
		} else {
			require.NotPanics(t, registerAgain)
		}
	})

	t.Run("return errors when blob unknown", func(t *testing.T) {
		fm := newOfflineFileManager(nil, "")

		_, err := fm.Open(ctx, blobName)
		require.ErrorContains(t, err, "unknown")

		err = fm.Upsert(ctx, blobName, time.Time{}, nil)
		require.ErrorContains(t, err, "unknown")
	})

	t.Run("open returns nil when file does not exist", func(t *testing.T) {
		dir := t.TempDir()

		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(nil, false, nil) // does not exist

		fm := newOfflineFileManager(blobStore, dir)
		fm.Register(blobName)

		f, err := fm.Open(ctx, blobName)
		require.NoError(t, err)
		require.Nil(t, f)
	})

	t.Run("open returns file on success", func(t *testing.T) {
		dir := t.TempDir()

		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(&storage.Blob{}, true, nil) // exists

		fm := newOfflineFileManager(blobStore, dir)
		fm.Register(blobName)

		f, err := fm.Open(ctx, blobName)
		require.NoError(t, err)
		require.NotNil(t, f)
	})

	t.Run("upsert returns error when blob upsert fails", func(t *testing.T) {
		dir := t.TempDir()
		ctrl := gomock.NewController(t)

		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("broken"))

		fm := newOfflineFileManager(blobStore, dir)
		fm.Register(blobName)

		err := fm.Upsert(ctx, blobName, time.Now(), openerSuccess())
		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(dir, blobName))
	})

	t.Run("upsert resets offline file when write fails", func(t *testing.T) {
		dir := t.TempDir()
		ctrl := gomock.NewController(t)

		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, false, nil). // does not exist
			Times(2)
		blobStore.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			Times(2)

		fm := newOfflineFileManager(blobStore, dir)
		fm.Register(blobName)

		// Force initialization
		_, err := fm.Open(ctx, blobName)
		require.NoError(t, err)
		require.True(t, fm.files[blobName].initialized.Load() == 1)

		err = fm.Upsert(ctx, blobName, time.Now(), openerErrorOnRead())
		require.NoError(t, err)
		require.False(t, fm.files[blobName].initialized.Load() == 1)

		// Force initialization
		_, err = fm.Open(ctx, blobName)
		require.NoError(t, err)
		require.True(t, fm.files[blobName].initialized.Load() == 1)

		// The '2nd' open corresponds to writing file to disk.
		err = fm.Upsert(ctx, blobName, time.Now(), openerErrorOnOpen(2))
		require.NoError(t, err)
		require.False(t, fm.files[blobName].initialized.Load() == 1)
	})

	t.Run("upsert returns error when opening src file fails", func(t *testing.T) {
		dir := t.TempDir()
		ctrl := gomock.NewController(t)

		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)

		fm := newOfflineFileManager(blobStore, dir)
		fm.Register(blobName)

		// The '1st' open corresponds to writing file to blob store.
		err := fm.Upsert(ctx, blobName, time.Now(), openerErrorOnOpen(1))
		require.ErrorContains(t, err, "blob in db")
	})

	t.Run("upsert creates blob and file on success", func(t *testing.T) {
		dir := t.TempDir()
		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		fm := newOfflineFileManager(blobStore, dir)
		fm.Register(blobName)

		err := fm.Upsert(ctx, blobName, time.Now(), openerSuccess())
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(dir, blobName))
	})
}

func TestOfflineFileLazyInit(t *testing.T) {
	ctx := context.Background()
	blobName := "fake"
	iterations := 3

	setup := func(t *testing.T) (*offlineFile, *blobDatastoreMocks.MockDatastore) {
		dir := t.TempDir()
		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)

		offlineFile := newOfflineFile(dir, blobStore, blobName)

		return offlineFile, blobStore
	}

	t.Run("init once on success", func(t *testing.T) {
		offlineFile, blobStore := setup(t)

		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(&storage.Blob{}, true, nil)

		for range iterations {
			f, _, err := offlineFile.Open(ctx)
			require.NoError(t, err)
			utils.IgnoreError(f.Close)
		}
	})

	t.Run("init once if blob does not exist", func(t *testing.T) {
		offlineFile, blobStore := setup(t)

		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(nil, false, nil)

		for range iterations {
			f, _, err := offlineFile.Open(ctx)
			require.NoError(t, err)
			utils.IgnoreError(f.Close)
		}
	})

	t.Run("init many times if error creating local on disk", func(t *testing.T) {
		offlineFile, blobStore := setup(t)

		// Error reading blob.
		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(nil, false, errors.New("broken")).
			Times(iterations)

		for range iterations {
			f, _, err := offlineFile.Open(ctx)
			require.NoError(t, err)
			utils.IgnoreError(f.Close)
		}
	})

	t.Run("init again on reset", func(t *testing.T) {
		offlineFile, blobStore := setup(t)

		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(&storage.Blob{}, true, nil).
			Times(2)

		for range iterations {
			f, _, err := offlineFile.Open(ctx)
			require.NoError(t, err)
			utils.IgnoreError(f.Close)
		}

		offlineFile.reset()

		f, _, err := offlineFile.Open(ctx)
		require.NoError(t, err)
		utils.IgnoreError(f.Close)
	})
}

func openerSuccess() offlineFileOpenerFunc {
	contents := "success!"
	return func() (io.ReadCloser, int64, error) {
		reader := io.NopCloser(strings.NewReader(contents))
		return reader, int64(len(contents)), nil
	}
}

// openerErrorOnOpen returns an opener that will trigger an
// error on file open on or after a certain number of attempts.
func openerErrorOnOpen(attempts int) offlineFileOpenerFunc {
	count := 1
	return func() (io.ReadCloser, int64, error) {
		if count >= (attempts) {
			return nil, 0, errors.New("open failed")
		}
		count++
		return openerSuccess()()
	}
}

// openerErrorOnRead returns an opener that will trigger an
// error when a read is attempted.
func openerErrorOnRead() offlineFileOpenerFunc {
	return func() (io.ReadCloser, int64, error) {
		reader := io.NopCloser(&errorReader{})
		return reader, 0, nil
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
