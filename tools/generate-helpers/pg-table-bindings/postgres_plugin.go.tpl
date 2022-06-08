
package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

{{- if not .JoinTable }}
{{ template "copyObject" .Schema }}

{{ template "copyFrom" . }}
{{- end }}
