package vulndump

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/pkg/cache"
	"github.com/stackrox/scanner/pkg/repo2cpe"
	"github.com/stackrox/scanner/pkg/wellknownkeys"
	"github.com/stackrox/scanner/pkg/ziputil"
)

var (
	// This lets us do some basic validation on the dumps, since no dumps were created before this date.
	// Therefore, if a dump has a start time before this timestamp, its start timestamp MUST be the zero time,
	// and its end timestamp MUST be after this time.
	earliestDump = timeutil.MustParse(time.RFC3339, "2019-11-19T00:00:00Z")

	updateLockName = "update"
)

func validateAndLoadManifest(f io.ReadCloser) (*Manifest, error) {
	defer utils.IgnoreError(f.Close)
	var m Manifest
	err := json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, errors.Wrap(err, "decoding manifest")
	}
	// This is a "genesis" dump. This will only get loaded in CI, never in prod.
	// Make sure that the time goes from 0->after earliestDump.
	if m.Since.Before(earliestDump) {
		if !m.Since.IsZero() {
			return nil, errors.Errorf("invalid since time in manifest: %s; it is before our known earliest dump, but not the zero time!", m.Since)
		}
		if m.Until.Before(earliestDump) {
			return nil, errors.Errorf("invalid until time in manifest: %s; the dump is a genesis dump, but ends before our earliestDump time!", m.Until)
		}
	} else {
		// Not a genesis dump. This path will be hit during regular updates.
		// Make sure it starts after earliestDump->some time after that.
		if !m.Since.After(earliestDump) {
			return nil, errors.Errorf("unexpected dump: not a genesis dump, but starts before our earliest known dump (at %s)", m.Since)
		}
		if !m.Until.After(m.Since) {
			return nil, errors.Errorf("unexpected dump: m.Until (%s) is not after m.Since (%s)", m.Until, m.Since)
		}
	}
	return &m, nil
}

func determineWhetherToUpdate(db database.Datastore, manifest *Manifest) (bool, error) {
	val, err := db.GetKeyValue(wellknownkeys.VulnUpdateTimestampKey)
	if err != nil {
		return false, errors.Wrap(err, "getting last update timestamp from DB")
	}
	// If the val is empty, that means this is a first update.
	// That means that we MUST make sure that the dump is a genesis dump.
	if val == "" {
		if !manifest.Since.IsZero() {
			return false, errors.New("DB is empty, but this dump is NOT a genesis dump. We NEED to load a genesis dump first.")
		}
		// Nothing in the DB, and this is a genesis dump. Let's update.
		return true, nil
	}

	// Not a first update. We update only if the manifest contains updates from after the most recent
	// update in the DB.
	var dbTime time.Time
	if err := dbTime.UnmarshalText(bytes.TrimSpace([]byte(val))); err != nil {
		return false, errors.Wrapf(err, "invalid timestamp in DB: %q", val)
	}
	// Protect against missing data.
	if manifest.Since.After(dbTime) {
		return false, errors.Errorf("cannot update with manifest: its start time (%s) is after the DB update time (%s)", manifest.Since, dbTime)
	}
	log.Infof(`Comparing manifest's "until" of %s to last DB update of %s`, manifest.Until, dbTime)
	return manifest.Until.After(dbTime), nil
}

// LoadManifestFromDump validates and loads the manifest from the given zip file.
func LoadManifestFromDump(zipR *zip.ReadCloser) (*Manifest, error) {
	manifestFile, err := ziputil.OpenFile(zipR, ManifestFileName)
	if err != nil {
		return nil, errors.Wrap(err, "opening manifest file")
	}
	manifest, err := validateAndLoadManifest(manifestFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading/validating manifest")
	}
	return manifest, nil
}

// LoadOSVulnsFromDump loads the os vulns file from the dump into an in-memory slice.
func LoadOSVulnsFromDump(zipR *zip.ReadCloser) ([]database.Vulnerability, error) {
	osVulnsFile, err := ziputil.OpenFile(zipR, OSVulnsFileName)
	if err != nil {
		return nil, errors.Wrap(err, "opening OS vulns file")
	}
	defer utils.IgnoreError(osVulnsFile.Close)

	var vulns []database.Vulnerability
	if err := json.NewDecoder(osVulnsFile).Decode(&vulns); err != nil {
		return nil, errors.Wrap(err, "JSON decoding OS vulns")
	}
	return vulns, nil
}

func renew(sig *concurrency.Signal, db database.Datastore, interval time.Duration, expiration time.Time, instanceName string) {
	// Give a buffer for this instance to renew the lock
	expirationDuration := time.Until(expiration) - 10*time.Second
	for {
		select {
		case <-time.After(expirationDuration):
			gotLock, newExpiration := db.Lock(updateLockName, instanceName, interval, true)
			if !gotLock {
				owner, _, err := db.FindLock(updateLockName)
				if err != nil {
					log.Errorf("error finding lock: %v", err)
					return
				}
				log.Errorf("DB update lock could not be renewed because it has already been acquired by %q", owner)
				return
			}
			expirationDuration = time.Until(newExpiration) - 10*time.Second
		case <-sig.Done():
			db.Unlock(updateLockName, instanceName)
			return
		}
	}

}

// startVulnLoad determines if this scanner should perform a vulnerability update and performs the necessary setup.
// The returned function should be performed upon update completion.
func startVulnLoad(manifest *Manifest, db database.Datastore, updateInterval time.Duration, instanceName string) (bool, func(err error) error, error) {
	shouldUpdate, err := determineWhetherToUpdate(db, manifest)
	if err != nil {
		return false, nil, errors.Wrap(err, "determining whether to update")
	}
	if !shouldUpdate {
		log.Info("DB already contains all the vulns in the dump. Nothing to do here!")
		return false, nil, nil
	}
	log.Info("Running the update.")

	gotLock, expiration := db.Lock(updateLockName, instanceName, updateInterval, false)
	if !gotLock {
		owner, _, err := db.FindLock(updateLockName)
		if err != nil {
			return false, nil, err
		}
		log.Infof("DB update lock already acquired by %q", owner)
		return false, nil, nil
	}
	finishedSig := concurrency.NewSignal()
	go renew(&finishedSig, db, updateInterval, expiration, instanceName)

	return true, func(err error) error {
		defer finishedSig.Signal()

		if err != nil {
			return err
		}

		log.Info("Done inserting vulns into the DB")

		marshaledDumpTS, err := manifest.Until.MarshalText()
		// Really shouldn't happen because we literally just unmarshalled it.
		utils.Must(err)
		if err := db.InsertKeyValue(wellknownkeys.VulnUpdateTimestampKey, string(marshaledDumpTS)); err != nil {
			return errors.Wrap(err, "couldn't update timestamp key in DB")
		}
		return nil
	}, nil
}

func loadOSVulns(zipR *zip.ReadCloser, db database.Datastore) error {
	log.Info("Loading OS vulns...")
	osVulns, err := LoadOSVulnsFromDump(zipR)
	if err != nil {
		return err
	}
	log.Infof("Done loading OS vulns. There are %d vulns to insert into the DB", len(osVulns))

	if err := db.InsertVulnerabilities(osVulns); err != nil {
		return errors.Wrap(err, "inserting vulns into the DB")
	}
	log.Info("Done inserting OS vulns into the DB")
	return nil
}

func loadRHELv2Vulns(db database.Datastore, zipR *zip.ReadCloser, repoToCPE *repo2cpe.Mapping) error {
	log.Info("Loading RHELv2 vulns...")
	if repoToCPE != nil {
		if err := repoToCPE.LoadFromZip(zipR, RHELv2DirName); err != nil {
			return errors.Wrap(err, "loading repo-to-cpe file into memory")
		}
	}

	rhelv2Readers, err := ziputil.OpenFilesInDir(zipR, filepath.Join(RHELv2DirName, RHELv2VulnsSubDirName), ".json")
	if err != nil {
		return errors.Wrap(err, "opening file in zip file")
	}
	log.Info("Done loading RHELv2 vulns")

	for _, r := range rhelv2Readers {
		if err := insertRHELv2Vulns(db, r); err != nil {
			return errors.Wrap(err, "inserting RHELv2 vulns into DB")
		}
	}

	log.Info("Done inserting RHELv2 vulns into the DB")
	return nil
}

// insertRHELv2Vulns inserts the RHELv2 vulns from the given io.ReadCloser
// into the DB.
// The given reader is closed upon return.
func insertRHELv2Vulns(db database.Datastore, r *ziputil.ReadCloser) error {
	defer utils.IgnoreError(r.Close)

	var vulns RHELv2
	if err := json.NewDecoder(r).Decode(&vulns); err != nil {
		return errors.Wrapf(err, "decoding JSON from %s", r.Name)
	}

	if err := db.InsertRHELv2Vulnerabilities(vulns.Vulns); err != nil {
		return errors.Wrapf(err, "inserting RHELv2 vulns from %s into the DB", r.Name)
	}
	log.Debugf("Done inserting vulns from %q into DB", r.Name)

	return nil
}

// This loads application-level vulnerabilities.
// At the moment, this consists of vulnerabilities from NVD and K8s.
func loadApplicationUpdater(cache cache.Cache, manifest *Manifest, zipR *zip.ReadCloser) error {
	if cache != nil {
		updateTime := cache.GetLastUpdate()
		if !updateTime.IsZero() && !manifest.Until.After(updateTime) {
			return nil
		}
		if err := cache.LoadFromZip(zipR, cache.Dir()); err != nil {
			return errors.Wrapf(err, "couldn't update in mem copy of %s", cache.Dir())
		}
		cache.SetLastUpdate(manifest.Until)
	}
	return nil
}

// UpdateFromVulnDump updates the definitions (both in the DB and in the Application Caches) from the given zip file.
// Check the well_known_names.go file for the manifest of the ZIP file.
func UpdateFromVulnDump(zipPath string, db database.Datastore, updateInterval time.Duration, instanceName string, caches []cache.Cache, repoToCPE *repo2cpe.Mapping) error {
	log.Infof("Attempting to update from vuln dump at %q", zipPath)

	if filepath.Ext(zipPath) != ".zip" {
		return errors.Errorf("invalid path %q: only .zip files are supported", zipPath)
	}

	zipR, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Wrap(err, "opening zip file")
	}
	defer utils.IgnoreError(zipR.Close)

	log.Info("Loading manifest...")
	manifest, err := LoadManifestFromDump(zipR)
	if err != nil {
		return err
	}
	log.Info("Loaded manifest")

	if db != nil {
		performUpdate, finishFn, err := startVulnLoad(manifest, db, updateInterval, instanceName)
		if err != nil {
			return errors.Wrap(err, "error beginning vuln loading")
		}
		if performUpdate {
			if err := loadRHELv2Vulns(db, zipR, repoToCPE); err != nil {
				_ = finishFn(err)
				return errors.Wrap(err, "error loading RHEL vulns")
			}

			if err := loadOSVulns(zipR, db); err != nil {
				_ = finishFn(err)
				return errors.Wrap(err, "error loading OS vulns")
			}

			if err := finishFn(nil); err != nil {
				return errors.Wrap(err, "error ending vuln loading")
			}
		}
	}

	errorList := errorhelpers.NewErrorList("loading application-level caches")
	for _, appCache := range caches {
		if err := loadApplicationUpdater(appCache, manifest, zipR); err != nil {
			errorList.AddError(errors.Wrapf(err, "error loading into in-mem cache %q", appCache.Dir()))
		}
	}

	return errorList.ToError()
}
