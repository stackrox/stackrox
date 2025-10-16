package file

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	blobDatastoreMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWriteBlob(t *testing.T) {
	ctx := context.Background()
	blobName := "fake"

	t.Run("return error when reading blob fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(nil, false, errors.New("broken"))

		f := New(filepath.Join(t.TempDir(), "test.txt"))
		err := f.WriteBlob(ctx, blobStore, blobName)
		require.ErrorContains(t, err, "writing blob")
	})

	t.Run("return not exist error when blob not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Return(nil, false, nil)

		f := New(filepath.Join(t.TempDir(), "test.txt"))
		err := f.WriteBlob(ctx, blobStore, blobName)
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("file created and modified time updated on success", func(t *testing.T) {
		oneHourAgo := time.Now().Add(-1 * time.Hour)
		dir := t.TempDir()
		blob := &storage.Blob{}
		blob.SetModifiedTime(protocompat.ConvertTimeToTimestampOrNil(&oneHourAgo))

		ctrl := gomock.NewController(t)
		blobStore := blobDatastoreMocks.NewMockDatastore(ctrl)
		blobStore.EXPECT().Get(ctx, blobName, gomock.Any()).
			Do(func(_ context.Context, blobName string, writer io.Writer) {
				_, err := writer.Write([]byte("howdy"))
				require.NoError(t, err)
			}).
			Return(blob, true, nil)

		liveFilePath := filepath.Join(dir, "live.txt")
		f := New(liveFilePath)
		err := f.WriteBlob(ctx, blobStore, blobName)
		require.NoError(t, err)
		require.FileExists(t, liveFilePath)

		fi, err := os.Stat(liveFilePath)
		require.NoError(t, err)
		require.Equal(t, oneHourAgo.Hour(), fi.ModTime().Hour())

		data, err := os.ReadFile(liveFilePath)
		require.NoError(t, err)
		require.Equal(t, "howdy", string(data))
	})
}
