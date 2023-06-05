package initbundlerotation

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type InitBundleRotator struct {
	ticker    concurrency.RetryTicker
	k8sClient kubernetes.Interface
	namespace string
}

func (i InitBundleRotator) Start() error {
	i.ticker = concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		return 0, nil
	}, time.Second*10, wait.Backoff{})

	return i.ticker.Start()
}

func (i InitBundleRotator) Stop(err error) {
	i.Stop(err)
}

func (i InitBundleRotator) Notify(e common.SensorComponentEvent) {
	//TODO implement me
}

func (i InitBundleRotator) Capabilities() []centralsensor.SensorCapability {
	//TODO implement me
	return nil
}

func (i InitBundleRotator) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_InitBundleGenResponse:
		secretsYAML := m.InitBundleGenResponse.KubectlBundle
		var secrets []*v1.Secret
		err := yaml.Unmarshal(secretsYAML, secrets)
		if err != nil {
			return errors.Wrap(err, "unmarshalling kubectl init bundle YAMLs")
		}

		for _, secret := range secrets {
			_, err := i.k8sClient.CoreV1().Secrets(i.namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "could not update secret %s/%s", secret.GetNamespace(), secret.GetName())
			}
		}
	}

	return nil
}

func (i InitBundleRotator) ResponsesC() <-chan *central.MsgFromSensor {
	//TODO implement me
	panic("implement me")
}

var _ common.SensorComponent = (*InitBundleRotator)(nil)
