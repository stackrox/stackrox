{{- $ := . }}
{{- $pks := .Schema.LocalPrimaryKeys }}

{{- $singlePK := dict.nil }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{- end }}

package postgres

import (
    "context"

    "github.com/stackrox/rox/pkg/sac"
)

type PermissionChecker interface {
    CountAllowed(ctx context.Context) (bool, error)
    ExistsAllowed(ctx context.Context) (bool, error)
    GetAllowed(ctx context.Context) (bool, error)
{{- if not .JoinTable }}
    UpsertAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error)
    UpsertManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error)
    DeleteAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error)
{{- end }}

{{- if $singlePK }}
    GetIDsAllowed(ctx context.Context) (bool, error)
    GetManyAllowed(ctx context.Context) (bool, error)
    DeleteManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error)
{{- end }}
}
