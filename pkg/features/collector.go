package features

var (
	// CollectorEbpf is used to enable eBPF
	CollectorEbpf = registerFeature("Enable eBPF Data Collection", "ROX_COLLECTOR_EBPF", false)
)
