package kubelet

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/compliance/collection/utils"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/pointers"
	"github.com/stackrox/stackrox/pkg/set"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubelet/config/v1beta1"
)

var (
	clientSchema = scheme.Scheme
	decoder      = serializer.NewCodecFactory(clientSchema).UniversalDeserializer()
)

func init() {
	if err := v1beta1.SchemeBuilder.AddToScheme(clientSchema); err != nil {
		panic(err)
	}
}

func applyConfigDefaults(kc *v1beta1.KubeletConfiguration) {
	// --anonymous-auth
	kc.Authentication.Anonymous.Enabled = pointers.Bool(true)
	// --authentication-token-webhook
	kc.Authentication.Webhook.Enabled = pointers.Bool(false)
	// --authorization-mode
	kc.Authorization.Mode = v1beta1.KubeletAuthorizationModeAlwaysAllow
	// --read-only-port
	kc.ReadOnlyPort = 10255

	kc.StreamingConnectionIdleTimeout = metav1.Duration{Duration: 4 * time.Hour}
	kc.MakeIPTablesUtilChains = pointers.Bool(true)
	kc.EventRecordQPS = pointers.Int32(5)
}

// GatherKubelet gets the KubeletConfiguration
func GatherKubelet() (*standards.KubeletConfiguration, error) {
	var config, hostnameOverride string
	flagSet := utils.NewFlagSet("kubelet")
	flagSet.StringVar(&config, "config", "", "")
	flagSet.StringVar(&hostnameOverride, "hostname-override", "", "")

	if err := utils.ParseFlags(set.NewStringSet("kubelet"), flagSet); err != nil {
		return nil, err
	}
	if config == "" {
		return nil, errors.New("no configuration file argument found for kubelet config")
	}
	var configuration v1beta1.KubeletConfiguration
	applyConfigDefaults(&configuration)

	data, err := utils.ReadHostFile(config)
	if err != nil {
		return nil, errors.Wrapf(err, "reading file %q for kubelet", config)
	}
	_, _, err = decoder.Decode(data, nil, &configuration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode kubelet config file")
	}
	return &standards.KubeletConfiguration{
		KubeletConfiguration: &configuration,
		HostnameOverride:     hostnameOverride,
	}, nil
}
