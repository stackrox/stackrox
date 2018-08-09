package utils

import (
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type configStatus struct {
	accessed bool
	err      error
}

// KubeAPIServerConfig is the Kubernetes API Server Config
var KubeAPIServerConfig FlattenedConfig
var kubeAPIServerConfigStatus configStatus

// KubeSchedulerConfig is the Kubernetes Scheduler Config
var KubeSchedulerConfig FlattenedConfig
var kubeSchedulerConfigStatus configStatus

// KubeControllerManagerConfig is the Kubernetes Controller Manager Config
var KubeControllerManagerConfig FlattenedConfig
var kubeControllerManagerConfigStatus configStatus

// EtcdConfig is the Etcd Config
var EtcdConfig FlattenedConfig
var etcdConfigStatus configStatus

// KubeletConfig is the Kubelet Config
var KubeletConfig FlattenedConfig
var kubeletConfigStatus configStatus

// KubeProxyConfig is the Kubernetes Proxy Config
var KubeProxyConfig FlattenedConfig
var kubeProxyConfigStatus configStatus

// KubeFederationAPIServerConfig is the Kubernetes Federation API Server Config
var KubeFederationAPIServerConfig FlattenedConfig
var kubeFederationAPIServerConfigStatus configStatus

// KubeFederationControllerManagerConfig is the Kubernetes Controller Manager Config
var KubeFederationControllerManagerConfig FlattenedConfig
var kubeFederationControllerManagerConfigStatus configStatus

var kubeCommandExpansion = map[string]string{}

func renderKubeConfig(processes ...string) (FlattenedConfig, error) {
	pid, process, err := getProcessPID(processes)
	if err != nil {
		return nil, err
	}
	cmdLine, err := getCommandLine(pid)
	if err != nil {
		return nil, err
	}
	args := getCommandLineArgs(cmdLine, process)
	config := make(FlattenedConfig)
	// Populate the configuration with the arguments
	parseArgs(config, args, kubeCommandExpansion)
	return config, nil
}

// InitKubeAPIServerConfig initializes the API Server Config
func InitKubeAPIServerConfig() error {
	if !kubeAPIServerConfigStatus.accessed {
		var err error
		KubeAPIServerConfig, err = renderKubeConfig("kube-apiserver", "/hyperkube apiserver")
		kubeAPIServerConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeAPIServerConfigStatus.err
}

// GetKubeAPIServerConfig retrieves the API Server Config
func GetKubeAPIServerConfig() (FlattenedConfig, error) {
	err := InitKubeAPIServerConfig()
	return KubeAPIServerConfig, err
}

// InitKubeSchedulerConfig initializes the Scheduler Config
func InitKubeSchedulerConfig() error {
	if !kubeAPIServerConfigStatus.accessed {
		var err error
		KubeSchedulerConfig, err = renderKubeConfig("kube-scheduler", "/hyperkube scheduler")
		kubeSchedulerConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeSchedulerConfigStatus.err
}

// GetKubeSchedulerConfig retrieves the Scheduler Config
func GetKubeSchedulerConfig() (FlattenedConfig, error) {
	err := InitKubeSchedulerConfig()
	return KubeSchedulerConfig, err
}

// InitKubeControllerManagerConfig initializes the Controller Manager Config
func InitKubeControllerManagerConfig() error {
	if !kubeControllerManagerConfigStatus.accessed {
		var err error
		KubeControllerManagerConfig, err = renderKubeConfig("kube-controller-manager", "/hyperkube controller-manager")
		kubeControllerManagerConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeControllerManagerConfigStatus.err
}

// GetKubeControllerManagerConfig retrieves the Controller Manager Config
func GetKubeControllerManagerConfig() (FlattenedConfig, error) {
	err := InitKubeControllerManagerConfig()
	return KubeControllerManagerConfig, err
}

// InitEtcdConfig initializes the Etcd Config
func InitEtcdConfig() error {
	if !etcdConfigStatus.accessed {
		var err error
		EtcdConfig, err = renderKubeConfig("etcd")
		etcdConfigStatus = configStatus{accessed: true, err: err}
	}
	return etcdConfigStatus.err
}

// GetEtcdConfig retrieves the Etcd Config
func GetEtcdConfig() (FlattenedConfig, error) {
	err := InitEtcdConfig()
	return EtcdConfig, err
}

// InitKubeletConfig initializes the Kubelet Config
func InitKubeletConfig() error {
	if !kubeletConfigStatus.accessed {
		var err error
		KubeletConfig, err = renderKubeConfig("kubelet", "/hyperkube kubelet")
		kubeletConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeletConfigStatus.err
}

// GetKubeletConfig retrieves the Kubelet Config
func GetKubeletConfig() (FlattenedConfig, error) {
	err := InitKubeletConfig()
	return KubeletConfig, err
}

// InitKubeFederationAPIServerConfig initializes the Federation API Server Config
func InitKubeFederationAPIServerConfig() error {
	if !kubeFederationAPIServerConfigStatus.accessed {
		var err error
		KubeFederationAPIServerConfig, err = renderKubeConfig("federation-apiserver", "/hyperkube federation-apiserver")
		kubeFederationAPIServerConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeFederationAPIServerConfigStatus.err
}

// GetKubeFederationControllerManagerConfig retrieves the Federation Controller Manager Config
func GetKubeFederationControllerManagerConfig() (FlattenedConfig, error) {
	err := InitKubeFederationAPIServerConfig()
	return KubeFederationAPIServerConfig, err
}

// InitKubeFederationControllerManagerConfig initializes the Controller Manager Config
func InitKubeFederationControllerManagerConfig() error {
	if !kubeFederationControllerManagerConfigStatus.accessed {
		var err error
		KubeFederationControllerManagerConfig, err = renderKubeConfig("federation-controller-manager", "/hyperkube federation-controller-manager")
		kubeFederationControllerManagerConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeFederationControllerManagerConfigStatus.err
}

// GetKubeFederationAPIServerConfig retrieves the Federation API Server
func GetKubeFederationAPIServerConfig() (FlattenedConfig, error) {
	err := InitKubeFederationControllerManagerConfig()
	return KubeFederationControllerManagerConfig, err
}

// InitKubeProxyConfig initializes the Kube Proxy
func InitKubeProxyConfig() error {
	if !kubeProxyConfigStatus.accessed {
		var err error
		KubeProxyConfig, err = renderKubeConfig("kube-proxy", "/hyperkube proxy")
		kubeProxyConfigStatus = configStatus{accessed: true, err: err}
	}
	return kubeProxyConfigStatus.err
}
