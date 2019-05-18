package manager

import (
	"context"
	"io/ioutil"
	"sort"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deploymentenvs"
	"github.com/stackrox/rox/central/license/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/license/validator"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	storeRetryInterval = 5 * time.Second
	// should match license-volume path in central deployment.yaml and
	// data key name in values.yaml
	secretPath = "/var/run/secrets/stackrox.io/central-license/license.lic"
)

var (
	log = logging.LoggerForModule()

	ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy)))
)

func licenseStatusToMetadataStatus(status v1.LicenseInfo_Status) v1.Metadata_LicenseStatus {
	switch status {
	case v1.LicenseInfo_UNKNOWN:
		return v1.Metadata_NONE
	case v1.LicenseInfo_VALID:
		return v1.Metadata_VALID
	case v1.LicenseInfo_EXPIRED:
		return v1.Metadata_EXPIRED
	default:
		return v1.Metadata_INVALID
	}
}

type licenseData struct {
	licenseProto                  *licenseproto.License
	notValidBefore, notValidAfter time.Time
	licenseKey                    string
}

func (d *licenseData) getLicenseProto() *licenseproto.License {
	if d == nil {
		return nil
	}
	return d.licenseProto
}

type manager struct {
	dataStore datastore.DataStore
	validator validator.Validator

	mutex         sync.RWMutex
	licenses      map[string]*licenseData
	activeLicense *licenseData

	dirty map[*licenseData]struct{}

	interruptC chan struct{}
	stopSig    concurrency.Signal
	stoppedSig concurrency.Signal

	listener LicenseEventListener

	deploymentEnvsMgr deploymentenvs.Manager

	licenseStatus v1.Metadata_LicenseStatus
}

func newManager(dataStore datastore.DataStore, validator validator.Validator, deploymentEnvsMgr deploymentenvs.Manager) *manager {
	return &manager{
		dataStore: dataStore,
		validator: validator,

		dirty: make(map[*licenseData]struct{}),

		interruptC: make(chan struct{}, 1),
		stopSig:    concurrency.NewSignal(),

		deploymentEnvsMgr: deploymentEnvsMgr,
	}
}

func (m *manager) interrupt() bool {
	select {
	case m.interruptC <- struct{}{}:
		return true
	case <-m.stoppedSig.Done():
		return false
	default:
		// If the above two cases block, we are not stopped and could not write to the channel. Since the channel is
		// buffered, there already is an interrupt pending, so no need for an additional one.
		return true
	}
}

func (m *manager) Initialize(listener LicenseEventListener) (*licenseproto.License, error) {
	if m.licenses != nil {
		return nil, errors.New("license manager was already initialized")
	}
	m.licenses = make(map[string]*licenseData)

	if err := m.populateFromStoreNoLock(); err != nil {
		return nil, errors.Wrap(err, "could not populate licenses from store")
	}

	m.populateLicenseFromSecretNoLock()

	m.checkLicensesNoLock(false)

	if m.activeLicense == nil && m.licenseStatus != v1.Metadata_NONE {
		log.Infof("No valid license found, but invalid licenses exist. Retrying after getting valid deployment environment information...")
		m.checkLicensesNoLock(true)
	}

	// Only set the listener now to prevent any event delivery during initial license selection.
	m.listener = listener

	go m.run()

	if listener != nil {
		listener.OnInitialize(m, m.activeLicense.getLicenseProto())
	}

	m.deploymentEnvsMgr.RegisterListener(deploymentEnvListener{
		manager: m,
	})

	return m.activeLicense.getLicenseProto(), nil
}

func (m *manager) Stop() concurrency.Waitable {
	m.stopSig.Signal()
	return &m.stoppedSig
}

func (m *manager) populateLicenseFromSecretNoLock() {
	data, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return
	}
	license, err := m.decodeLicenseKey((string)(data))
	if err != nil {
		log.Errorf("Invalid license data in secret: %s", err)
		return
	}
	deploymentEnvsByClusterID := m.deploymentEnvsMgr.GetDeploymentEnvironmentsByClusterID(false)
	_, err = m.addLicenseNoLock(deploymentEnvsByClusterID, license, false)
	status, statusReason := statusFromError(err)
	if status == v1.LicenseInfo_VALID {
		log.Infof("License successfully imported from orchestrator secret")
	} else {
		log.Errorf("Imported license but not valid: %s: %s", status, statusReason)
	}
}

func (m *manager) populateFromStoreNoLock() error {
	storedLicenseKeys, err := m.dataStore.ListLicenseKeys(ctx)
	if err != nil {
		return err
	}

	m.importStoredKeysNoLock(storedLicenseKeys)
	return nil
}

func (m *manager) importStoredKeysNoLock(storedKeys []*storage.StoredLicenseKey) {
	var selected *licenseData
	for _, storedKey := range storedKeys {
		license, err := m.decodeLicenseKey(storedKey.GetLicenseKey())
		if err != nil {
			log.Errorf("Could not read license key from store: %v. The license key will be ignored.", err)
			continue
		}
		if license.licenseProto.GetMetadata().GetId() != storedKey.GetLicenseId() {
			log.Errorf("Stored license key data is corrupted: ID %q does not match ID %q of decoded license. The license key will be ignored.", license.licenseProto.GetMetadata().GetIssueDate(), storedKey.GetLicenseId())
			continue
		}

		if storedKey.GetSelected() {
			if selected != nil {
				log.Errorf("Stored license key data is corrupted: multiple licenses (%q and %q) are marked as selected. Will default to the first one.", selected.licenseProto.GetMetadata().GetId(), license.licenseProto.GetMetadata().GetId())
			} else {
				selected = license
			}
		}

		m.licenses[license.licenseProto.GetMetadata().GetId()] = license
	}

	m.activeLicense = selected
}

func (m *manager) run() {
	m.stoppedSig.Reset()
	defer m.stoppedSig.Signal()

	var nextEventTimer *time.Timer

	for !m.stopSig.IsDone() {
		timeutil.StopTimer(nextEventTimer)
		nextEventTimer = nil

		nextEventTS := m.checkLicenses()

		if err := m.updateStore(); err != nil {
			log.Errorf("Could not update license key store: %v. Retrying in %v", err, storeRetryInterval)
			retryTS := time.Now().Add(storeRetryInterval)
			if nextEventTS.IsZero() || retryTS.Before(nextEventTS) {
				nextEventTS = retryTS
			}
		}

		if !nextEventTS.IsZero() {
			nextEventTimer = time.NewTimer(time.Until(nextEventTS))
		}

		select {
		case <-m.stopSig.Done():
			log.Info("License manager is shutting down.")
			timeutil.StopTimer(nextEventTimer)
			return
		case <-timeutil.TimerC(nextEventTimer):
			nextEventTimer = nil
		case <-m.interruptC:
		}
	}
}

func (m *manager) checkLicenses() time.Time {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.checkLicensesNoLock(true)
}

func (m *manager) checkLicensesNoLock(waitForDeploymentEnvs bool) time.Time {
	deploymentEnvsByClusterID := m.deploymentEnvsMgr.GetDeploymentEnvironmentsByClusterID(waitForDeploymentEnvs)

	if m.activeLicense != nil {
		err := m.checkLicenseIsUsable(m.activeLicense, deploymentEnvsByClusterID)
		if err == nil {
			m.licenseStatus = v1.Metadata_VALID
			return m.activeLicense.notValidAfter
		}

		log.Warnf("Disabling active license %s: %v", m.activeLicense.licenseProto.GetMetadata().GetId(), err)
	}

	// Do the following in a loop to ensure that we try to make a new license active until we either have found a new
	// license that works, or have determined that we do not currently have a usable license.
	// Otherwise, if `makeLicenseActiveNoLock` does not succeed because the license has become invalid (e.g., just
	// expired), we might not deactive the old license.
	for {
		newActiveLicense, invalidStatus, nextEventTS := m.chooseUsableLicenseNoLock(deploymentEnvsByClusterID)

		_, _, licenseChanged := m.makeLicenseActiveNoLock(newActiveLicense, deploymentEnvsByClusterID)
		if licenseChanged || newActiveLicense == nil {
			if newActiveLicense != nil {
				log.Infof("Automatically selected new license %s, valid until %v", newActiveLicense.licenseProto.GetMetadata().GetId(), newActiveLicense.notValidAfter)
				m.licenseStatus = v1.Metadata_VALID
			} else {
				m.licenseStatus = invalidStatus
			}
			return nextEventTS
		}
	}
}

// chooseUsableLicenseNoLock returns the "best" available license, or nil if no usable license is available. The
// second return value indicates the timestamp when we should next check for an available license (this could either
// be the expiration time of the chosen license, or the next time a license that is not yet valid becomes valid).
func (m *manager) chooseUsableLicenseNoLock(deploymentEnvsByClusterID map[string][]string) (*licenseData, v1.Metadata_LicenseStatus, time.Time) {
	var bestCandidate *licenseData

	var nextActivationTS time.Time
	now := time.Now()

	// invalidStatus is the most precise metadata license status of any invalid license.
	invalidStatus := v1.Metadata_NONE

	for _, license := range m.licenses {
		// Calculate the nearest `notValidBefore` timestamp that is in the future, regardless of why the license
		// is not valid (conditions might change, so we should always re-check once a license becomes valid time-wise).
		if now.Before(license.notValidBefore) && (nextActivationTS.IsZero() || license.notValidBefore.Before(nextActivationTS)) {
			nextActivationTS = license.notValidBefore
		}

		if err := m.checkLicenseIsUsable(license, deploymentEnvsByClusterID); err != nil {
			licenseStatus, _ := statusFromError(err)
			thisStatus := licenseStatusToMetadataStatus(licenseStatus)
			if thisStatus > invalidStatus {
				invalidStatus = thisStatus
			}
			continue
		}

		// For now, only select the license which is valid for the longest time.
		if bestCandidate == nil || license.notValidAfter.After(bestCandidate.notValidAfter) {
			bestCandidate = license
		}
	}

	if bestCandidate != nil {
		return bestCandidate, invalidStatus, bestCandidate.notValidAfter
	}
	return nil, invalidStatus, nextActivationTS
}

func (m *manager) getLicenseInfoNoLock(license *licenseData, deploymentEnvsByClusterID map[string][]string) *v1.LicenseInfo {
	if license == nil {
		return nil
	}

	licenseInfo := &v1.LicenseInfo{
		License: license.licenseProto,
		Active:  license == m.activeLicense,
	}
	licenseInfo.Status, licenseInfo.StatusReason = statusFromError(m.checkLicenseIsUsable(license, deploymentEnvsByClusterID))
	return licenseInfo
}

func (m *manager) markDirtyNoLock(license *licenseData) {
	if license != nil {
		m.dirty[license] = struct{}{}
	}
}

func (m *manager) makeLicenseActiveNoLock(newLicense *licenseData, deploymentEnvsByClusterID map[string][]string) (newLicenseInfo, oldLicenseInfo *v1.LicenseInfo, changed bool) {
	newLicenseInfo = m.getLicenseInfoNoLock(newLicense, deploymentEnvsByClusterID)

	oldLicense := m.activeLicense
	if oldLicense == newLicense {
		oldLicenseInfo = newLicenseInfo
		return
	}

	oldLicenseInfo = m.getLicenseInfoNoLock(oldLicense, deploymentEnvsByClusterID)
	if newLicenseInfo != nil {
		if newLicenseInfo.GetStatus() != v1.LicenseInfo_VALID {
			return // new license is not valid, so we cannot change it
		}
	}

	m.activeLicense = newLicense

	changed = true
	if newLicenseInfo != nil {
		newLicenseInfo.Active = true
	}
	if oldLicenseInfo != nil {
		oldLicenseInfo.Active = false
	}

	if m.listener != nil {
		m.listener.OnActiveLicenseChanged(newLicenseInfo, oldLicenseInfo)
	}

	m.markDirtyNoLock(oldLicense)
	m.markDirtyNoLock(newLicense)

	m.interrupt()

	return
}

func (m *manager) toStoredKeyNoLock(license *licenseData) *storage.StoredLicenseKey {
	return &storage.StoredLicenseKey{
		LicenseKey: license.licenseKey,
		LicenseId:  license.licenseProto.GetMetadata().GetId(),
		Selected:   license == m.activeLicense,
	}
}

func (m *manager) updateStore() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.dirty) == 0 {
		return nil
	}

	toUpsert := make([]*storage.StoredLicenseKey, 0, len(m.dirty))

	for dirtyLicense := range m.dirty {
		toUpsert = append(toUpsert, m.toStoredKeyNoLock(dirtyLicense))
	}
	m.dirty = make(map[*licenseData]struct{})

	return m.dataStore.UpsertLicenseKeys(ctx, toUpsert)
}

func (m *manager) GetActiveLicense() *licenseproto.License {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.activeLicense == nil {
		return nil
	}
	return m.activeLicense.licenseProto
}

func (m *manager) GetAllLicenses() []*v1.LicenseInfo {
	var allLicenses []*v1.LicenseInfo

	deploymentEnvsByClusterID := m.deploymentEnvsMgr.GetDeploymentEnvironmentsByClusterID(true)

	concurrency.WithRLock(&m.mutex, func() {
		log.Debugf("GetAllLicenses, active license is %v", m.activeLicense)
		allLicenses = make([]*v1.LicenseInfo, 0, len(m.licenses))

		for _, license := range m.licenses {
			log.Debugf("Processing license %v", license)
			allLicenses = append(allLicenses, m.getLicenseInfoNoLock(license, deploymentEnvsByClusterID))
		}
	})

	sort.Slice(allLicenses, func(i, j int) bool {
		return allLicenses[i].GetLicense().GetMetadata().GetId() < allLicenses[j].GetLicense().GetMetadata().GetId()
	})

	return allLicenses
}

func (m *manager) AddLicenseKey(licenseKey string, activate bool) (*v1.LicenseInfo, error) {
	license, err := m.decodeLicenseKey(licenseKey)
	if err != nil {
		return nil, errors.Wrap(err, "decoding license key")
	}

	return m.addLicense(license, activate)
}

func (m *manager) addLicense(license *licenseData, activate bool) (*v1.LicenseInfo, error) {
	defer m.interrupt()

	deploymentEnvsByClusterID := m.deploymentEnvsMgr.GetDeploymentEnvironmentsByClusterID(true)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.addLicenseNoLock(deploymentEnvsByClusterID, license, activate)
}

func (m *manager) addLicenseNoLock(deploymentEnvsByClusterID map[string][]string, license *licenseData, activate bool) (*v1.LicenseInfo, error) {
	// We accept licenses that violate constraints (the environment might change), but not those that are expired.
	licenseErr := m.checkLicenseIsUsable(license, deploymentEnvsByClusterID)
	if status, _ := statusFromError(licenseErr); status == v1.LicenseInfo_EXPIRED {
		return nil, licenseErr
	}

	id := license.licenseProto.GetMetadata().GetId()
	oldLicense := m.licenses[id]
	m.licenses[id] = license

	delete(m.dirty, oldLicense)
	m.dirty[license] = struct{}{}

	// If we're replacing the old license, we definitely want to activate the replacement. Also
	// try activating it if we don't have a currently active license (note: these conditions might
	// collapse into one if there is no active license and we are not replacing a license, but that is
	// perfectly fine).
	activate = activate || m.activeLicense == oldLicense || m.activeLicense == nil

	var newLicenseInfo *v1.LicenseInfo
	if licenseErr == nil && activate {
		newLicenseInfo, _, _ = m.makeLicenseActiveNoLock(license, deploymentEnvsByClusterID)
	} else {
		newLicenseInfo = m.getLicenseInfoNoLock(license, deploymentEnvsByClusterID)
	}

	mdStatusFromLicense := licenseStatusToMetadataStatus(newLicenseInfo.GetStatus())
	if mdStatusFromLicense > m.licenseStatus {
		m.licenseStatus = mdStatusFromLicense
	}

	return newLicenseInfo, nil
}

func (m *manager) decodeLicenseKey(licenseKey string) (*licenseData, error) {
	licenseProto, err := m.validator.ValidateLicenseKey(licenseKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not validate license key")
	}

	nvb, err := types.TimestampFromProto(licenseProto.GetRestrictions().GetNotValidBefore())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert NotValidBefore timestamp")
	}

	nva, err := types.TimestampFromProto(licenseProto.GetRestrictions().GetNotValidAfter())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert NotValidAfter timestamp")
	}

	return &licenseData{
		licenseProto:   licenseProto,
		notValidBefore: nvb,
		notValidAfter:  nva,
		licenseKey:     licenseKey,
	}, nil
}

func (m *manager) checkLicenseIsUsable(license *licenseData, deploymentEnvsByClusterID map[string][]string) error {
	// First check time-independent constraints. We do not want to say "not yet valid" for a license that won't
	// be usable anyway.
	if err := m.checkConstraints(license.licenseProto.GetRestrictions(), deploymentEnvsByClusterID); err != nil {
		return err
	}

	if time.Now().Before(license.notValidBefore) {
		return notYetValidError(license.notValidBefore)
	}

	if time.Now().After(license.notValidAfter) {
		return expiredError(license.notValidAfter)
	}
	return nil
}

func (m *manager) checkConstraints(restr *licenseproto.License_Restrictions, deploymentEnvsByClusterID map[string][]string) error {
	if err := checkDeploymentEnvironmentRestrictions(restr, deploymentEnvsByClusterID); err != nil {
		return err
	}

	// TODO: Enforce online licenses

	if !restr.GetNoBuildFlavorRestriction() {
		if sliceutils.StringFind(restr.GetBuildFlavors(), buildinfo.BuildFlavor) == -1 {
			return errors.Errorf("licenseData cannot be used with build flavor %s", buildinfo.BuildFlavor)
		}
	}
	return nil
}

func (m *manager) SelectLicense(id string) (*v1.LicenseInfo, error) {
	deploymentEnvsByClusterID := m.deploymentEnvsMgr.GetDeploymentEnvironmentsByClusterID(true)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	license := m.licenses[id]
	if license == nil {
		return nil, errors.Errorf("invalid license ID %q", id)
	}

	newLicenseInfo, _, _ := m.makeLicenseActiveNoLock(license, deploymentEnvsByClusterID)
	return newLicenseInfo, nil
}

func (m *manager) GetLicenseStatus() v1.Metadata_LicenseStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.licenseStatus
}
