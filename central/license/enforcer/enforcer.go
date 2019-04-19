package enforcer

import (
	"os"
	"time"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/license/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	// enforcementTimeBuffer specifies how long to wait before actually restarting. This doesn't need to be super long,
	// just make sure that the response is delivered to the client before we restart.
	enforcementTimeBuffer = 1 * time.Second

	// managerGracePeriod specifies how long to wait for the license manager to shut down.
	managerGracePeriod = 2 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// New creates a new license enforcement event listener.
func New(restartingFlag *concurrency.Flag) manager.LicenseEventListener {
	return &enforcer{
		restartingFlag: restartingFlag,
	}
}

type enforcer struct {
	licenseMgr             manager.LicenseManager
	initialLicenseWasValid bool
	restartTimer           *time.Timer
	restartingFlag         *concurrency.Flag
}

func (e *enforcer) OnInitialize(mgr manager.LicenseManager, initialLicense *licenseproto.License) {
	e.licenseMgr = mgr
	e.initialLicenseWasValid = initialLicense != nil
}

func (e *enforcer) OnActiveLicenseChanged(newLicenseInfo, oldLicenseInfo *v1.LicenseInfo) {
	if e.licenseMgr == nil {
		log.Panicf("License enforcer got triggered due to a license change but had not been initialized!")
	}

	log.Debugf("License change detected: %+v vs %+v", newLicenseInfo, oldLicenseInfo)

	validNow := newLicenseInfo.GetStatus() == v1.LicenseInfo_VALID
	if validNow != e.initialLicenseWasValid {
		if validNow {
			log.Infof("Valid license detected. Central will restart in %v for the change to take effect", enforcementTimeBuffer)
		} else {
			log.Infof("No valid license is available. Central will restart in %v for the change to take effect", enforcementTimeBuffer)
		}
		e.restartTimer = time.AfterFunc(enforcementTimeBuffer, e.enforce)
		e.restartingFlag.Set(true)
	} else if e.restartTimer != nil {
		if e.restartTimer.Stop() {
			log.Info("Restart was aborted.")
		}
		e.restartTimer = nil
		e.restartingFlag.Set(false)
	}
}

func (e *enforcer) enforce() {
	log.Info("Central restart due to license state change is going into effect now.")
	log.Infof("Shutting down license manager ...")
	if !concurrency.WaitWithTimeout(e.licenseMgr.Stop(), managerGracePeriod) {
		log.Warnf("License manager did not stop within %v", managerGracePeriod)
	}
	log.Infof("Closing databases ...")
	globaldb.Close()
	log.Info("Central is restarting.")
	os.Exit(0)
}
