package flags

import (
	"errors"
	"io/ioutil"
	"os"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/license"
)

var (
	errInvalidLicense = errors.New("invalid license data or file not found")
)

const (
	// LicenseUsage provides usage information for license flags defined by the struct in this package.
	LicenseUsage = "license data or filename (default: none, - to read stdin)"
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

func isValidLicense(val string) bool {
	_, _, err := license.ParseLicenseKey(val)
	return err == nil
}

func readLicenseFromStdin() ([]byte, error) {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "failed to read from stdin")
	}
	return data, nil
}

func tryReadLicenseFromFile(filename string) ([]byte, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return contents, nil
}

// Set implements the Value interface.
func (v *LicenseVar) Set(val string) error {
	if val == "" {
		v.Data = nil
		return nil
	}
	var data []byte
	var err error
	if isValidLicense(val) {
		data = []byte(val)
	} else if val == "-" {
		data, err = readLicenseFromStdin()
		if err != nil {
			return err
		}
	} else {
		data, err = tryReadLicenseFromFile(val)
		if err != nil {
			return err
		}
	}

	if data == nil {
		return errInvalidLicense
	}
	*v.Data = data
	return nil
}
