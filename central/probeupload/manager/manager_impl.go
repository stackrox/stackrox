package manager

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"path"
	"strings"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	blobstore "github.com/stackrox/rox/central/blob/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

const (
	rootBlobPathPrefix = `/offline/probe-uploads/`
	rootBlobPathRegex  = `/offline/probe-uploads/.+`

	defaultFreeStorageThreshold = 1 << 30 // 1 GB
	metadataSizeOverhead        = 16384   // 16KB of overhead for blob metadata and indexes etc.
)

var (
	log = logging.LoggerForModule()

	administrationSAC = sac.ForResource(resources.Administration)

	blobReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
)

type manager struct {
	rootDir              string
	blobStore            blobstore.Datastore
	freeStorageThreshold int64
}

func newManager(datastore blobstore.Datastore) *manager {
	return &manager{
		rootDir:              rootBlobPathPrefix,
		blobStore:            datastore,
		freeStorageThreshold: defaultFreeStorageThreshold,
	}
}

func (m *manager) getAllProbeBlobs() ([]string, error) {
	q := search.NewQueryBuilder().AddRegexes(search.BlobName, rootBlobPathRegex).ProtoQuery()
	blobs, err := m.blobStore.SearchIDs(blobReadAccessCtx, q)
	if err != nil {
		return nil, err
	}
	return blobs, nil
}

// Print a warning for each unrecognized entry.
func (m *manager) checkRootDir() error {
	probeBlobs, err := m.getAllProbeBlobs()
	if err != nil {
		return errors.Wrap(err, "could not read probe uploads from blobstore")
	}

	for _, blobName := range probeBlobs {
		moduleVer := strings.TrimPrefix(blobName, rootBlobPathPrefix)
		if !probeupload.IsValidFilePath(moduleVer) {
			log.Warnf("Unexpected non-module-version entry %q in probe upload blobs", blobName)
			continue
		}
	}
	return nil
}

func (m *manager) Initialize() error {
	return m.checkRootDir()
}

func (m *manager) getProbeUploadPath(file string) string {
	return path.Join(m.rootDir, file)
}

func (m *manager) loadBlob(ctx context.Context, file string) (io.ReadCloser, int64, error) {
	if !probeupload.IsValidFilePath(file) {
		return nil, 0, errors.Errorf("%q is not a valid probe file name", file)
	}

	uploadPath := m.getProbeUploadPath(file)
	buf, blob, exists, err := m.blobStore.GetBlobWithDataInBuffer(ctx, uploadPath)
	if err != nil || !exists {
		return nil, 0, err
	}
	return io.NopCloser(buf), blob.GetLength(), err
}

func (m *manager) getFileInfo(ctx context.Context, file string) (*v1.ProbeUploadManifest_File, error) {
	if !probeupload.IsValidFilePath(file) {
		return nil, errors.Errorf("%q is not a valid probe file name", file)
	}
	blob, exists, err := m.blobStore.GetMetadata(ctx, m.getProbeUploadPath(file))
	if err != nil || !exists {
		return nil, err
	}

	crc32Data := []byte(blob.GetChecksum())
	if len(crc32Data) != 4 {
		return nil, errors.Errorf("probe %s does not have a valid CRC-32 checksum (%d bytes)", file, len(crc32Data))
	}

	crc32 := binary.BigEndian.Uint32(crc32Data)

	return &v1.ProbeUploadManifest_File{
		Name:  file,
		Size_: blob.GetLength(),
		Crc32: crc32,
	}, nil
}

func (m *manager) GetExistingProbeFiles(ctx context.Context, files []string) ([]*v1.ProbeUploadManifest_File, error) {
	if ok, err := administrationSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var result []*v1.ProbeUploadManifest_File
	for _, file := range files {
		fi, err := m.getFileInfo(ctx, file)
		if err != nil {
			return nil, err
		}
		if fi != nil {
			result = append(result, fi)
		}
	}
	return result, nil
}

func (m *manager) StoreFile(ctx context.Context, file string, data io.Reader, size int64, crc32Sum uint32) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if !probeupload.IsValidFilePath(file) {
		return errors.Errorf("invalid file name %q", file)
	}

	// When using external databases, Postgres space cannot be calculated.
	if !env.ManagedCentral.BooleanSetting() && !pgconfig.IsExternalDatabase() {
		requiredBytes := uint64(size) + uint64(metadataSizeOverhead) + uint64(m.freeStorageThreshold)
		if freeBytes, err := availableBytes(); err == nil && uint64(freeBytes) < requiredBytes {
			return errors.Errorf("only %d bytes left on database, not storing probes to avoid impacting database health", freeBytes)
		}
	}

	verifyingReader := ioutils.NewCRC32ChecksumReader(io.LimitReader(data, size), crc32.IEEETable, crc32Sum)
	checksumBytes := binenc.BigEndian.EncodeUint32(crc32Sum)
	b := &storage.Blob{
		Name:         path.Join(rootBlobPathPrefix, file),
		Checksum:     string(checksumBytes),
		Length:       size,
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
	}

	if err := m.blobStore.Upsert(ctx, b, verifyingReader); err != nil {
		return errors.Wrapf(err, "writing probe data blob %s", file)
	}

	if err := verifyingReader.Close(); err != nil {
		return errors.Wrap(err, "error closing probe data reader (possible checksum violation)")
	}

	return nil
}

func (m *manager) LoadProbe(ctx context.Context, file string) (io.ReadCloser, int64, error) {
	return m.loadBlob(ctx, file)
}

func (m *manager) IsAvailable(_ context.Context) (bool, error) {
	blobs, err := m.getAllProbeBlobs()
	if err != nil {
		return false, err
	}
	return len(blobs) > 0, nil
}

func availableBytes() (int64, error) {
	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return 0, errors.Wrap(err, "Could not parse postgres config")
	}

	return pgadmin.GetRemainingCapacity(dbConfig)
}
