package datastore

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// GetID generates risk ID from risk ubject ID (e.g. deployment ID) and type.
func GetID(subjectID string, subjectType storage.RiskSubjectType) (string, error) {
	if stringutils.AllNotEmpty(subjectID, subjectType.String()) {
		return fmt.Sprintf("%s:%s", strings.ToLower(subjectType.String()), subjectID), nil
	}
	return "", errors.New("cannot build risk ID")
}

// RiskSubjectType returns enum of supplied subject type string.
func RiskSubjectType(subjectType string) (storage.RiskSubjectType, error) {
	value, found := storage.RiskSubjectType_value[strings.ToUpper(subjectType)]
	if !found {
		return storage.RiskSubjectType_UNKNOWN, errors.New("unknown subject type")
	}

	return storage.RiskSubjectType(value), nil
}
