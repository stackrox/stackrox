package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetTestSingleKeyStruct returns filled TestSingleKeyStruct
func GetTestSingleKeyStruct() *storage.TestSingleKeyStruct {
	tsks := &storage.TestSingleKeyStruct{}
	tsks.SetKey(uuid.NewDummy().String())
	tsks.SetName("name")
	tsks.SetStringSlice([]string{
		"slice1", "slice2",
	})
	tsks.SetBool(true)
	tsks.SetUint64(16)
	tsks.SetInt64(32)
	tsks.SetFloat(4.56)
	tsks.SetLabels(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	tsks.SetTimestamp(protocompat.GetProtoTimestampFromSecondsAndNanos(
		1645640515,
		0))

	tsks.SetEnum(storage.TestSingleKeyStruct_ENUM1)
	tsks.SetEnums([]storage.TestSingleKeyStruct_Enum{
		storage.TestSingleKeyStruct_ENUM1,
		storage.TestSingleKeyStruct_ENUM2,
	})
	return tsks
}
