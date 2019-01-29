package docker

import (
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/docker/go-units"
)

// The following structs are copied from 17.03-ce for Unix builds with embedded fields as exported
// This avoids importing daemon which is not compatible with other instantiations of GRPC
// and causes lots of build issues

// CommonTLSOptions copied from config.go
type CommonTLSOptions struct {
	CAFile   string `json:"tlscacert,omitempty"`
	CertFile string `json:"tlscert,omitempty"`
	KeyFile  string `json:"tlskey,omitempty"`
}

// LogConfig copied from config.go
type LogConfig struct {
	Type   string            `json:"log-driver,omitempty"`
	Config map[string]string `json:"log-opts,omitempty"`
}

// CommonBridgeConfig copied from config.go
type CommonBridgeConfig struct {
	Iface     string `json:"bridge,omitempty"`
	FixedCIDR string `json:"fixed-cidr,omitempty"`
}

// CommonUnixBridgeConfig copied from config_unix.go
type CommonUnixBridgeConfig struct {
	DefaultIP                   net.IP `json:"ip,omitempty"`
	IP                          string `json:"bip,omitempty"`
	DefaultGatewayIPv4          net.IP `json:"default-gateway,omitempty"`
	DefaultGatewayIPv6          net.IP `json:"default-gateway-v6,omitempty"`
	InterContainerCommunication bool   `json:"icc,omitempty"`
}

// BridgeConfig copied from config_unix.go
type BridgeConfig struct {
	CommonBridgeConfig
	CommonUnixBridgeConfig

	EnableIPv6          bool   `json:"ipv6,omitempty"`
	EnableIPTables      bool   `json:"iptables,omitempty"`
	EnableIPForward     bool   `json:"ip-forward,omitempty"`
	EnableIPMasq        bool   `json:"ip-masq,omitempty"`
	EnableUserlandProxy bool   `json:"userland-proxy,omitempty"`
	UserlandProxyPath   string `json:"userland-proxy-path,omitempty"`
	FixedCIDRv6         string `json:"fixed-cidr-v6,omitempty"`
}

// CommonConfig copied from config.go
type CommonConfig struct {
	AuthorizationPlugins   []string            `json:"authorization-plugins,omitempty"` // AuthorizationPlugins holds list of authorization plugins
	AutoRestart            bool                `json:"-"`
	Context                map[string][]string `json:"-"`
	DisableBridge          bool                `json:"-"`
	DNS                    []string            `json:"dns,omitempty"`
	DNSOptions             []string            `json:"dns-opts,omitempty"`
	DNSSearch              []string            `json:"dns-search,omitempty"`
	ExecOptions            []string            `json:"exec-opts,omitempty"`
	GraphDriver            string              `json:"storage-driver,omitempty"`
	GraphOptions           []string            `json:"storage-opts,omitempty"`
	Labels                 []string            `json:"labels,omitempty"`
	Mtu                    int                 `json:"mtu,omitempty"`
	Pidfile                string              `json:"pidfile,omitempty"`
	RawLogs                bool                `json:"raw-logs,omitempty"`
	Root                   string              `json:"graph,omitempty"`
	SocketGroup            string              `json:"group,omitempty"`
	TrustKeyPath           string              `json:"-"`
	CorsHeaders            string              `json:"api-cors-header,omitempty"`
	EnableCors             bool                `json:"api-enable-cors,omitempty"`
	LiveRestoreEnabled     bool                `json:"live-restore,omitempty"`
	ClusterStore           string              `json:"cluster-store,omitempty"`
	ClusterOpts            map[string]string   `json:"cluster-store-opts,omitempty"`
	ClusterAdvertise       string              `json:"cluster-advertise,omitempty"`
	MaxConcurrentDownloads *int                `json:"max-concurrent-downloads,omitempty"`
	MaxConcurrentUploads   *int                `json:"max-concurrent-uploads,omitempty"`
	ShutdownTimeout        int                 `json:"shutdown-timeout,omitempty"`
	Debug                  bool                `json:"debug,omitempty"`
	Hosts                  []string            `json:"hosts,omitempty"`
	LogLevel               string              `json:"log-level,omitempty"`
	TLS                    bool                `json:"tls,omitempty"`
	TLSVerify              bool                `json:"tlsverify,omitempty"`

	CommonTLSOptions

	SwarmDefaultAdvertiseAddr string `json:"swarm-default-advertise-addr"`
	MetricsAddress            string `json:"metrics-addr"`

	LogConfig
	BridgeConfig
	registry.ServiceOptions

	ValuesSet map[string]interface{}

	Experimental bool `json:"experimental"`
}

// CommonUnixConfig copied from config_common_unix.go
type CommonUnixConfig struct {
	ExecRoot       string                   `json:"exec-root,omitempty"`
	ContainerdAddr string                   `json:"containerd,omitempty"`
	Runtimes       map[string]types.Runtime `json:"runtimes,omitempty"`
	DefaultRuntime string                   `json:"default-runtime,omitempty"`
}

// Config copied from config_unix.go
type Config struct {
	CommonConfig

	// These fields are common to all unix platforms.
	CommonUnixConfig

	// Fields below here are platform specific.
	CgroupParent         string                   `json:"cgroup-parent,omitempty"`
	EnableSelinuxSupport bool                     `json:"selinux-enabled,omitempty"`
	RemappedRoot         string                   `json:"userns-remap,omitempty"`
	Ulimits              map[string]*units.Ulimit `json:"default-ulimits,omitempty"`
	CPURealtimePeriod    int64                    `json:"cpu-rt-period,omitempty"`
	CPURealtimeRuntime   int64                    `json:"cpu-rt-runtime,omitempty"`
	OOMScoreAdjust       int                      `json:"oom-score-adjust,omitempty"`
	Init                 bool                     `json:"init,omitempty"`
	InitPath             string                   `json:"init-path,omitempty"`
	SeccompProfile       string                   `json:"seccomp-profile,omitempty"`
	NoNewPrivileges      bool                     `json:"no-new-privileges,omitempty"`
	IpcMode              string                   `json:"default-ipc-mode,omitempty"`
	// ResolvConf is the path to the configuration of the host resolver
	ResolvConf string `json:"resolv-conf,omitempty"`
}

// DockerCommandExpansion tranforms the short versions of the dockerd command line into their full names
var DockerCommandExpansion = map[string]string{
	"b": "bridge",
	"D": "debug",
	"G": "group",
	"H": "host",
	"l": "log-level",
	"p": "pidfile",
	"s": "storage-driver",
}
