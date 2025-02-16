package main

import (
	"strings"

	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var (
	tableToProtoMessage = map[string]protocompat.Message{}
)

var knownUnhandledBuckets = set.NewStringSet()

// getProtoMessage retrieves the proto message for the given table name.
func getProtoMessage(tableName string) (protocompat.Message, bool) {
	msg, ok := tableToProtoMessage[tableName]
	if !ok && strings.HasPrefix(tableName, "network_flows_v2") {
		msg, ok = tableToProtoMessage["network_flows_v2"]
	}
	return msg, ok
}

func normalizeName(name string) string {
	name = strings.TrimPrefix(name, "*")
	name = strings.TrimPrefix(name, "storage.")
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, ".", " ")
	return name
}

func init() {
	allProtoTypeMap := make(map[string]protocompat.Message)
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		fullName := string(mt.Descriptor().FullName())
		if !strings.HasPrefix(fullName, "storage.") {
			return true
		}
		allProtoTypeMap[normalizeName(fullName)] = mt.New().Interface()
		return true
	})

	allSchema := schema.GetAllSchemas()
	for _, s := range allSchema {
		gatherTableMap(allProtoTypeMap, s)
	}
}

// gatherTableMap maps schema table names to their corresponding proto messages.
func gatherTableMap(protoTypeMap map[string]protocompat.Message, schema *walker.Schema) {
	mt := protoTypeMap[normalizeName(schema.Type)]
	if mt == nil {
		log.Errorf("Failed to find proto for table %s: %s", schema.Table, schema.Type)
		return
	}
	tableToProtoMessage[schema.Table] = mt
	for _, child := range schema.Children {
		gatherTableMap(protoTypeMap, child)
	}
}
