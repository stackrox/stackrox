{{- define "convertTest" }}
{{- $schema := . }}
    func Test{{$schema.TypeName}}Serialization(t *testing.T) {
        obj := &storage.{{$schema.TypeName}}{}
        assert.NoError(t, testutils.FullInit(obj, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
        m, err := Convert{{$schema.TypeName}}FromProto(obj)
        assert.NoError(t, err)
        conv, err := Convert{{$schema.TypeName}}ToProto(m)
        assert.NoError(t, err)
        assert.Equal(t, obj, conv)
    }
{{- end}}
package schema

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

{{- template "convertTest" .Schema }}
