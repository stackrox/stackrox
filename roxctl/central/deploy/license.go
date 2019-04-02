package deploy

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/stackrox/rox/pkg/license"
)

var (
	errInvalidLicense = errors.New("invalid license data or file not found")
)

type licenseVar struct {
	data *[]byte
}

func (licenseVar) Type() string {
	return "license"
}

func (v licenseVar) String() string {
	if v.data == nil {
		return ""
	}
	return "<license data>"
}

func (v *licenseVar) Set(val string) error {
	if val == "" {
		v.data = nil
		return nil
	}
	if val == "-" {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		*v.data = data
		return nil
	}
	_, _, err := license.ParseLicenseKey(val)
	if err == nil {
		data := ([]byte)(val)
		*v.data = data
		return nil
	}
	data, ioErr := ioutil.ReadFile(val)
	if ioErr == nil {
		*v.data = data
		return nil
	}
	if os.IsNotExist(ioErr) {
		return errInvalidLicense
	}
	return ioErr
}
