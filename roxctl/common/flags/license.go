package flags

const (
	// LicenseUsage provides usage information for license flags defined by the struct in this package.
	LicenseUsage = "(DEPRECATED) license is no longer needed"
)

// LicenseVar represents a set-table variable for the license file.
type LicenseVar struct {
	Data *[]byte
}

// Type implements the Value interface.
func (LicenseVar) Type() string {
	return "license"
}

// String implements the Value interface.
func (v LicenseVar) String() string {
	if v.Data == nil || len(*v.Data) == 0 {
		return ""
	}
	return "<license data>"
}

// Set implements the Value interface.
func (v *LicenseVar) Set(val string) error {
	return nil
}
