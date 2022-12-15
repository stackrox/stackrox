package utils

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/groups"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// Value has been taken from:
	//	https://github.com/stackrox/stackrox/blob/6a702b26d66dcc2236a742907809071249187070/central/group/datastore/validate.go#L13
	groupIDPrefix = "io.stackrox.authz.group."
	// Value has been taken from:
	//	https://github.com/stackrox/stackrox/blob/1bd8c26d4918c3b530ad4fd713244d9cf71e786d/migrator/migrations/m_105_to_m_106_group_id/migration.go#L134
	groupMigratedIDPrefix = "io.stackrox.authz.group.migrated."
)

// ValidationErrorCode specifies the error which occurred during verifying a key/value pair in the groups bucket.
type ValidationErrorCode int

func (v ValidationErrorCode) String() string {
	return [...]string{
		"unset", "wrong-key-format", "invalid-uuid-in-key", "marshal-proto-message-error", "invalid-group-proto-message",
	}[v]
}

const (
	// UnsetErrorCode default value.
	UnsetErrorCode ValidationErrorCode = iota
	wrongKeyFormat
	invalidUUID
	errorMarshalProtoMessage
	invalidGroupProto
)

// ValidGroupKeyValuePair validates the key/value pair stored within a bolt.DB grous bucket.
// It will return true if the pair is valid.
// It will return false if the pair is invalid, and a ValidationErrorCode specifying why the entry is seen as invalid.
func ValidGroupKeyValuePair(k, v []byte) (bool, ValidationErrorCode) {
	key := string(k)

	// Ensure the key has the correct prefix for a group.
	if !strings.HasPrefix(key, groupIDPrefix) && !strings.HasPrefix(key, groupMigratedIDPrefix) {
		return false, wrongKeyFormat
	}

	// Ensure the key contains a valid UUID after trimming the prefix.
	// Note that the order is important, as trimming group ID prefix with a migrated ID would leave a .migrated.
	key = strings.TrimPrefix(key, groupMigratedIDPrefix)
	key = strings.TrimPrefix(key, groupIDPrefix)
	_, err := uuid.FromString(key)
	if err != nil {
		return false, invalidUUID
	}

	// Ensure that the value can be unmarshalled to a group proto message.
	var group storage.Group
	if err := group.Unmarshal(v); err != nil {
		return false, errorMarshalProtoMessage
	}

	// Ensure that the group is a valid group.
	if err := groups.ValidateGroup(&group, true); err != nil {
		return false, invalidGroupProto
	}

	return true, UnsetErrorCode
}
