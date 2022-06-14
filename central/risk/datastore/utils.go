package datastore

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// GetID generates risk ID from risk subject ID (e.g. deployment ID) and type.
func GetID(subjectID string, subjectType storage.RiskSubjectType) (string, error) {
	if stringutils.AllNotEmpty(subjectID, subjectType.String()) {
		return fmt.Sprintf("%s:%s", strings.ToLower(subjectType.String()), subjectID), nil
	}
	return "", errors.New("cannot build risk ID")
}

// GetIDParts returns subject type and subject ID from risk ID.
func GetIDParts(riskID string) (storage.RiskSubjectType, string, error) {
	idParts := strings.SplitN(riskID, ":", 2)
	if len(idParts) != 2 {
		return storage.RiskSubjectType_UNKNOWN, "", errors.New("cannot extract id parts")
	}
	subjectType, err := SubjectType(idParts[0])
	return subjectType, idParts[1], err
}

// SubjectType returns enum of supplied subject type string.
func SubjectType(subjectType string) (storage.RiskSubjectType, error) {
	value, found := storage.RiskSubjectType_value[strings.ToUpper(subjectType)]
	if !found {
		return storage.RiskSubjectType_UNKNOWN, errors.New("unknown subject type")
	}

	return storage.RiskSubjectType(value), nil
}
