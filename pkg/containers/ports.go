package containers

// Exposure levels for ports.
const (
	Internal = `internal`
	Node     = `node`
	External = `external`
)

// IncreasedExposureLevel returns whether the new level carries increased exposure.
func IncreasedExposureLevel(old, new string) bool {
	switch old {
	case "":
		return true
	case Internal:
		return new == Node || new == External
	case Node:
		return new == External
	default:
		return false
	}
}
