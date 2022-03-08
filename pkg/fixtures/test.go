package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// GetTestSingleKeyStruct returns filled TestSingleKeyStruct
func GetTestSingleKeyStruct() *storage.TestSingleKeyStruct {
	return &storage.TestSingleKeyStruct{
		Key:  "key1",
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

// GetTestMultiKeyStruct returns filled TestMultiKeyStruct
func GetTestMultiKeyStruct() *storage.TestMultiKeyStruct {
	return &storage.TestMultiKeyStruct{
		Key1: "key1",
		Key2: "key2",
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
		Enum: storage.TestMultiKeyStruct_ENUM1,
		Enums: []storage.TestMultiKeyStruct_Enum{
			storage.TestMultiKeyStruct_ENUM1,
			storage.TestMultiKeyStruct_ENUM2,
		},
	}
}
