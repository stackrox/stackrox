package walker

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stretchr/testify/assert"
)

type LineageInfo struct {
	ParentUID          uint32 `protobuf:"varint,1,opt,name=parent_uid,json=parentUid,proto3" json:"parent_uid,omitempty"`
	ParentExecFilePath string `protobuf:"bytes,2,opt,name=parent_exec_file_path,json=parentExecFilePath,proto3" json:"parent_exec_file_path,omitempty"`
}

type TestSignal struct {
	Name        string         `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	LineageInfo []*LineageInfo `protobuf:"bytes,2,rep,name=lineage_info,json=lineageInfo,proto3" json:"lineage_info,omitempty"`
}

type TestIndicator struct {
	ID     string      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk,id"`
	Signal *TestSignal `protobuf:"bytes,2,opt,name=signal,proto3" json:"signal,omitempty"`
}

func TestArrayStrategy(t *testing.T) {
	schema := Walk(
		reflect.TypeOf(&TestIndicator{}),
		"test_indicators",
		WithNoSerialized(),
		WithRepeatedFieldStrategies(map[string]string{
			"signal.lineage_info": "array",
		}),
	)

	// Verify array columns were created
	var foundParentUID, foundParentExecFilePath bool
	for _, f := range schema.Fields {
		// Column names preserve the CamelCase from the walker context
		if f.ColumnName == "Signal_LineageInfo_parentuid" {
			foundParentUID = true
			assert.Equal(t, postgres.ArrayColumn, f.DataType)
			assert.Equal(t, "int4[]", f.SQLType)
			assert.Equal(t, "[]uint32", f.Type)
			assert.Equal(t, "[]uint32", f.ModelType)
		}
		if f.ColumnName == "Signal_LineageInfo_parentexecfilepath" {
			foundParentExecFilePath = true
			assert.Equal(t, postgres.ArrayColumn, f.DataType)
			assert.Equal(t, "text[]", f.SQLType)
			assert.Equal(t, "[]string", f.Type)
			assert.Equal(t, "[]string", f.ModelType)
		}
	}

	assert.True(t, foundParentUID, "Expected signal_lineageinfo_parentuid array column")
	assert.True(t, foundParentExecFilePath, "Expected signal_lineageinfo_parentexecfilepath array column")

	// Verify no child table was created for lineage_info
	assert.Empty(t, schema.Children, "Should not create child tables when using array strategy")
}

func TestByteasStrategy(t *testing.T) {
	schema := Walk(
		reflect.TypeOf(&TestIndicator{}),
		"test_indicators",
		WithNoSerialized(),
		// No strategy specified, defaults to bytea
	)

	// Verify bytea column was created
	var foundBytea bool
	for _, f := range schema.Fields {
		// Column names preserve CamelCase from walker context
		if f.ColumnName == "Signal_LineageInfo" && f.DataType == postgres.MessageBytes {
			foundBytea = true
			assert.Equal(t, "bytea", f.SQLType)
			assert.Equal(t, "[]byte", f.Type)
		}
	}

	assert.True(t, foundBytea, "Expected Signal_LineageInfo bytea column")
}

func TestBuildProtoFieldPath(t *testing.T) {
	tests := []struct {
		name         string
		ctx          walkerContext
		protoName    string
		expectedPath string
	}{
		{
			name:         "root level field",
			ctx:          walkerContext{column: ""},
			protoName:    "LineageInfo",
			expectedPath: "lineageinfo",
		},
		{
			name:         "nested field",
			ctx:          walkerContext{column: "signal"},
			protoName:    "LineageInfo",
			expectedPath: "signal.lineageinfo",
		},
		{
			name:         "deeply nested field",
			ctx:          walkerContext{column: "signal_metadata"},
			protoName:    "Tags",
			expectedPath: "signal.metadata.tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := buildProtoFieldPath(tt.ctx, tt.protoName)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestGoTypeToArraySQLType(t *testing.T) {
	tests := []struct {
		goType      reflect.Type
		expectedSQL string
	}{
		{reflect.TypeOf(""), "text[]"},
		{reflect.TypeOf(int32(0)), "int4[]"},
		{reflect.TypeOf(uint32(0)), "int4[]"},
		{reflect.TypeOf(int64(0)), "int8[]"},
		{reflect.TypeOf(uint64(0)), "int8[]"},
		{reflect.TypeOf(true), "bool[]"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedSQL, func(t *testing.T) {
			sqlType := goTypeToArraySQLType(tt.goType)
			assert.Equal(t, tt.expectedSQL, sqlType)
		})
	}
}

func TestGoTypeToArrayGoType(t *testing.T) {
	tests := []struct {
		goType     reflect.Type
		expectedGo string
	}{
		{reflect.TypeOf(""), "[]string"},
		{reflect.TypeOf(int32(0)), "[]int32"},
		{reflect.TypeOf(uint32(0)), "[]uint32"},
		{reflect.TypeOf(int64(0)), "[]int64"},
		{reflect.TypeOf(uint64(0)), "[]uint64"},
		{reflect.TypeOf(true), "[]bool"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedGo, func(t *testing.T) {
			goType := goTypeToArrayGoType(tt.goType)
			assert.Equal(t, tt.expectedGo, goType)
		})
	}
}
