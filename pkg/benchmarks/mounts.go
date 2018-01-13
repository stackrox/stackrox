package benchmarks

// BenchmarkMounts are the mounts required for running benchmarks
var BenchmarkMounts = []string{
	"/etc:/host/etc:ro",
	"/lib/systemd:/host/lib/systemd:ro",
	"/usr/bin:/host/usr/bin:ro",
	"/usr/lib/systemd:/host/usr/lib/systemd:ro",
	"/var/lib:/host/var/lib:ro",
	"/var/log/audit:/host/var/log/audit:ro",
	"/var/run/docker.sock:/host/var/run/docker.sock",
}

const (
	// BenchmarkCommand is the command to run the benchmark container
	BenchmarkCommand = "benchmarks"
	// BenchmarkBootstrapCommand is the command to run the benchmark bootstrap container
	BenchmarkBootstrapCommand = "benchmark-bootstrap"
)
