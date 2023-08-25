package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetTestSingleKeyStruct returns filled TestSingleKeyStruct
func GetTestSingleKeyStruct() *storage.TestSingleKeyStruct {
	return &storage.TestSingleKeyStruct{
		Key:  uuid.NewDummy().String(),
		Name: "name",
		StringSlice: []string{
			"slice1", "slice2",
		},
		Bool:   true,
		Uint64: 16,
		Int64:  32,
		Float:  4.56,
		Labels: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Timestamp: &types.Timestamp{
			Seconds: 1645640515,
			Nanos:   0,
		},
		Enum: storage.TestSingleKeyStruct_ENUM1,
		Enums: []storage.TestSingleKeyStruct_Enum{
			storage.TestSingleKeyStruct_ENUM1,
			storage.TestSingleKeyStruct_ENUM2,
		},
	}
}
