package registrymirrors

import (
	"testing"
	"time"

	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
)

// TODO: Add tests, test delay with a mock and test the FS write with a temp file
func TestUpdateConfig(t *testing.T) {
	svc := NewDelayedService(NewFileService("/Users/dcaravel/dev/stackrox/stackrox-mirror-filesys/dignore/registries.conf"))
	log.Debugf("start")

	icspRules := []*operatorV1Alpha1.ImageContentSourcePolicy{
		{},
	}

	idmsRules := []*configV1.ImageDigestMirrorSet{
		{},
	}

	svc.UpdateConfig(nil, nil, nil)
	svc.UpdateConfig(nil, nil, nil)
	svc.UpdateConfig(icspRules, idmsRules, nil)
	time.Sleep(time.Millisecond * 300)
	svc.UpdateConfig(nil, nil, nil)
	svc.UpdateConfig(nil, nil, nil)
	svc.UpdateConfig(nil, nil, nil)
	svc.UpdateConfig(nil, nil, nil)

	time.Sleep(time.Second * 1)

	// test file exists
}
