package flags

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/license"
	"github.com/stackrox/rox/roxctl/common/mode"
)

var (
	errInvalidLicense = errors.New("invalid license data or file not found")
)

const (
	// LicenseUsage provides usage information for license flags defined by the struct in this package.
	LicenseUsage = "license data or filename (default: none, - to read stdin)"
	// LicenseUsageInteractive is the usage information that will be shown for the interactive prompt.
	LicenseUsageInteractive = "license data or filename (`-` for multiline input)"

	// minSignatureBytes is the minimum number of bytes for the base64-decoded part right of the dot
	// in a license key or filename string in order for us to assume it actually is a license key.
	// We use ECDSA256/384 signing keys, so the minimum length for a DER-encoded signature is actually
	// 72 bytes, but 32 bytes should be more than enough to distinguish it from file names.
	minSignatureBytes = 32
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
	_, sig, err := license.ParseLicenseKey(val)
	return err == nil && len(sig) >= minSignatureBytes
}

func readLicenseFromStdin() ([]byte, error) {
	if mode.IsInInteractiveMode() {
		_, _ = fmt.Fprintln(os.Stderr, "Reading license data from terminal. Press Enter followed by Ctrl+D to mark end of input.")
	}
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
	} else {
		// An `@` character (which is not a valid base64 encoded character) may be used to circumvent
		// autodetection and mark the argument as a file name (or stdin reference).
		val = strings.TrimPrefix(val, "@")
		if val == "-" {
			data, err = readLicenseFromStdin()
		} else {
			data, err = tryReadLicenseFromFile(val)
		}
	}

	if err != nil {
		return err
	}
	if data == nil {
		return errInvalidLicense
	}

	*v.Data = data
	return nil
}
