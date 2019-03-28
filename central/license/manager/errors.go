package manager

import (
	"fmt"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

type licenseError interface {
	error
	Status() v1.LicenseInfo_Status
}

type notYetValidError time.Time

func (e notYetValidError) Error() string {
	return fmt.Sprintf("license is not valid for another %v", time.Time(e).Sub(time.Now()))
}

func (notYetValidError) Status() v1.LicenseInfo_Status {
	return v1.LicenseInfo_NOT_YET_VALID
}

type expiredError time.Time

func (e expiredError) Error() string {
	return fmt.Sprintf("license expired %v ago", time.Now().Sub(time.Time(e)))
}

func (expiredError) Status() v1.LicenseInfo_Status {
	return v1.LicenseInfo_EXPIRED
}

type revokedError string

func (e revokedError) Error() string {
	msg := "license has been revoked"
	if e != "" {
		msg += " for the following reason: " + string(e)
	}
	return msg
}

func (revokedError) Status() v1.LicenseInfo_Status {
	return v1.LicenseInfo_REVOKED
}

func statusFromError(err error) (v1.LicenseInfo_Status, string) {
	if err == nil {
		return v1.LicenseInfo_VALID, ""
	}

	if licenseErr, ok := err.(licenseError); ok {
		return licenseErr.Status(), licenseErr.Error()
	}
	return v1.LicenseInfo_OTHER, err.Error()
}
