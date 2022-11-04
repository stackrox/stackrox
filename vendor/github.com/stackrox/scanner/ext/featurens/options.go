package featurens

// DetectorOptions are the options used for base-OS detectors.
type DetectorOptions struct {
	UncertifiedRHEL bool
}

// GetUncertifiedRHEL returns "true" if UncertifiedRHEL is true; "false" otherwise.
func (do *DetectorOptions) GetUncertifiedRHEL() bool {
	if do == nil {
		return false
	}

	return do.UncertifiedRHEL
}
