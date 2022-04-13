package gatherer

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	entityDataStore "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	"github.com/stackrox/stackrox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/utils"
)

func loadBundledExternalSrcs(networkEntityDS entityDataStore.EntityDataStore) error {
	// Extract the bundle to temp dir.
	checksumFile, dataFile, err := extractBundle(defaultexternalsrcs.BundledZip)
	if err != nil {
		return errors.Wrap(err, "extracting external networks bundle")
	}

	newChecksum, err := os.ReadFile(checksumFile)
	if err != nil {
		return errors.Wrapf(err, "reading bundled external networks checksum from %q", checksumFile)
	}

	var localChecksum []byte
	_, err = os.Open(defaultexternalsrcs.LocalChecksumFile)
	if os.IsExist(err) {
		localChecksum, err = os.ReadFile(defaultexternalsrcs.LocalChecksumFile)
		if err != nil {
			return errors.Wrapf(err, "reading local external networks checksum from %q", defaultexternalsrcs.LocalChecksumFile)
		}
	}

	if bytes.Equal(localChecksum, newChecksum) {
		return nil
	}

	data, err := os.ReadFile(dataFile)
	if err != nil {
		return errors.Wrap(err, "reading new external networks data")
	}

	entities, err := defaultexternalsrcs.ParseProviderNetworkData(data)
	if err != nil {
		return err
	}

	log.Infof("Successfully loaded %d external networks", len(entities))

	lastSeenIDs, err := loadStoredDefaultExtSrcsIDs(networkEntityDS)
	if err != nil {
		return err
	}

	inserted, err := updateInStorage(networkEntityDS, lastSeenIDs, entities...)
	if err != nil {
		return errors.Wrapf(err, "updated %d/%d networks", len(inserted), len(entities))
	}

	log.Infof("Found %d external networks in DB. Successfully stored %d/%d new external networks", len(lastSeenIDs), len(inserted), len(entities))

	// Update checksum only if all the pulled data is successfully written.
	if err := writeChecksumLocally(newChecksum); err != nil {
		return err
	}

	newIDs := set.NewStringSet()
	for _, entity := range entities {
		newIDs.Add(entity.GetInfo().GetId())
	}

	if err := removeOutdatedNetworks(networkEntityDS, lastSeenIDs.Difference(newIDs).AsSlice()...); err != nil {
		return errors.Wrap(err, "removing outdated default external networks")
	}
	return nil
}

func extractBundle(src string) (string, string, error) {
	zipR, err := zip.OpenReader(src)
	if err != nil {
		return "", "", errors.Wrapf(err, "couldn't open file %q as zip", src)
	}
	defer utils.IgnoreError(zipR.Close)

	tmpPath, err := os.MkdirTemp("", defaultexternalsrcs.SubDir)
	if err != nil {
		return "", "", err
	}

	err = os.MkdirAll(tmpPath, 0755)
	if err != nil {
		return "", "", errors.Wrap(err, "creating temp sub-directory for external networks")
	}

	for _, zipF := range zipR.File {
		if zipF.FileInfo().IsDir() {
			continue
		}

		reader, err := zipF.Open()
		if err != nil {
			return "", "", errors.Wrap(err, "opening reader")
		}
		defer utils.IgnoreError(reader.Close)

		file, err := os.Create(path.Join(tmpPath, zipF.FileInfo().Name()))
		if err != nil {
			return "", "", errors.Wrap(err, "creating external networks temp file")
		}

		_, err = io.Copy(file, reader)
		if err != nil {
			return "", "", errors.Wrap(err, "copying external networks zip out")
		}

	}
	return path.Join(tmpPath, defaultexternalsrcs.ChecksumFileName), path.Join(tmpPath, defaultexternalsrcs.DataFileName), nil
}
