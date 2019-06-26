package netutil

type (
	// EphemeralPortConfidence is a numeric value indicating the confidence that a port is an ephemeral port. The higher
	// the value, the more confident we are that the port is ephemeral.
	EphemeralPortConfidence int
)

const (
	// AbsoluteEphemeralPortConfidenceThreshold is the confidence threshold when a port can be assumed
	// to be ephemeral in the absence of the port of a remote counterpart to compare it to.
	AbsoluteEphemeralPortConfidenceThreshold EphemeralPortConfidence = 3
)

// IsEphemeralPort returns a confidence value indicating how confident we are that a given port is ephemeral.
// This is not an absolute value, but rather can be used to determine which endpoint of a connection (local/remote)
// is the client by determining for which port there is the higher confidence that it is ephemeral.
func IsEphemeralPort(port uint16) EphemeralPortConfidence {
	switch {
	// IANA range
	case port >= 49152:
		return 4
	// Modern Linux kernel range
	case port >= 32768:
		return 3
	// FreeBSD (partial) + Windows <=XP range
	case port >= 1025 && port <= 5000:
		return 2
	// FreeBSD
	case port == 1024:
		return 1
	}
	// not ephemeral according to any range
	return 0
}
