package settingswatch

import (
	"os"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/admission-control/common"
	"github.com/stackrox/rox/sensor/admission-control/manager"
)

const (
	settingsFileName     = `settings.pb`
	settingsTempFileName = `.settings-temp.pb`
)

var (
	settingsPath     = filepath.Join(common.TempStoragePath, settingsFileName)
	settingsTempPath = filepath.Join(common.TempStoragePath, settingsTempFileName)
)

// RunSettingsPersister runs a component that reads persisted settings from a mounted directory once, and stores every
// settings update to that same directory. This allows a hot start after a container restart.
func RunSettingsPersister(mgr manager.Manager) error {
	if err := os.Remove(settingsTempPath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "clearing out temporary settings file %s", settingsPath)
	}

	p := &persister{
		ctx:              mgr.Stopped(),
		outC:             mgr.SettingsUpdateC(),
		settingsStreamIt: mgr.SettingsStream().Iterator(false),
	}

	go p.run()
	return nil
}

type persister struct {
	ctx              concurrency.Waitable
	settingsStreamIt concurrency.ValueStreamIter[*sensor.AdmissionControlSettings]
	outC             chan<- *sensor.AdmissionControlSettings
}

func (p *persister) run() {
	settings, err := p.loadExisting()
	if err != nil {
		log.Warnf("Error loading initial admission control settings: %v", err)
	} else if settings != nil {
		select {
		case <-p.ctx.Done():
			return
		case p.outC <- settings:
		}
		log.Infof("Detected and propagated initial admission controller settings from temporary storage, timestamp: %v", settings.GetTimestamp())
	}

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.settingsStreamIt.Done():
			p.settingsStreamIt = p.settingsStreamIt.TryNext()
			if err := p.persistCurrent(); err != nil {
				log.Errorf("Failed to persist updated settings: %v", err)
			}
		}
	}
}

func (p *persister) loadExisting() (*sensor.AdmissionControlSettings, error) {
	bytes, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "loading initial admission control settings from %s", settingsPath)
	}

	var settings sensor.AdmissionControlSettings
	if err := proto.Unmarshal(bytes, &settings); err != nil {
		return nil, errors.Wrapf(err, "unmarshaling initial admission control settings from %s", settingsPath)
	}

	return &settings, nil
}

func (p *persister) persistCurrent() error {
	settings := p.settingsStreamIt.Value()
	if settings == nil {
		if err := os.Remove(settingsPath); err != nil {
			return errors.Wrapf(err, "removing existing settings path %s", settingsPath)
		}
		return nil
	}

	bytes, err := proto.Marshal(settings)
	if err != nil {
		return errors.Wrap(err, "marshaling settings proto")
	}

	defer func() { _ = os.Remove(settingsTempPath) }()
	if err := os.WriteFile(settingsTempPath, bytes, 0640); err != nil {
		return errors.Wrapf(err, "writing to temporary file %s", settingsTempPath)
	}

	if err := os.Rename(settingsTempPath, settingsPath); err != nil {
		return errors.Wrapf(err, "atomically copying new settings file %s to %s", settingsTempPath, settingsPath)
	}

	return nil
}
