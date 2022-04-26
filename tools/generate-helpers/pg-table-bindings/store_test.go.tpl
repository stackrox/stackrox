
{{define "paramList"}}{{$name := .TrimmedType|lowerCamelCase}}{{range $idx, $pk := .Schema.LocalPrimaryKeys}}{{if $idx}}, {{end}}{{$pk.Getter $name}}{{end}}{{end}}

{{- $ := . }}
{{- $name := .TrimmedType|lowerCamelCase }}

{{- $namePrefix := .Table|upperCamelCase}}

//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped) }}
    "github.com/stackrox/rox/pkg/sac"{{- end }}
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type {{$namePrefix}}StoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func Test{{$namePrefix}}Store(t *testing.T) {
	suite.Run(t, new({{$namePrefix}}StoreSuite))
}

func (s *{{$namePrefix}}StoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}
}

func (s *{{$namePrefix}}StoreSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *{{$namePrefix}}StoreSuite) TestStore() {
    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped)}}
    ctx := sac.WithAllAccess(context.Background())
    {{- else -}}
    ctx := context.Background()
    {{- end }}

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)
	store := New(ctx, pool)

	{{$name}} := &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	found{{.TrimmedType|upperCamelCase}}, exists, err := store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})

    {{if not .JoinTable -}}
    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped)}}
    withNoAccessCtx := sac.WithNoAccess(ctx)
    {{- end }}
	{{- if .Obj.IsDirectlyScoped }}
	withAccessToDifferentNsCtx:= sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(targetResource),
			sac.ClusterScopeKeys({{ $name | .Obj.GetClusterID }}),
			sac.NamespaceScopeKeys("unknown ns"),
	))
	withAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(targetResource),
			sac.ClusterScopeKeys({{ $name | .Obj.GetClusterID }}),
			{{- if .Obj.IsNamespaceScope }}
			sac.NamespaceScopeKeys({{ $name | .Obj.GetNamespace }}),
			{{- end }}
	))
	withAccessToClusterCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys({{ $name | .Obj.GetClusterID }}),
	))
	withNoAccessToClusterCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys("unknown cluster"),
	))
	{{- end }}

	s.NoError(store.Upsert(ctx, {{$name}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	{{$name}}Count, err := store.Count(ctx)
	s.NoError(err)
	s.Equal({{$name}}Count, 1)

    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) }}
    {{$name}}Count, err = store.Count(withNoAccessCtx)
    s.NoError(err)
    s.Zero({{$name}}Count)
    {{- end }}

	{{$name}}Exists, err := store.Exists(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True({{$name}}Exists)
	s.NoError(store.Upsert(ctx, {{$name}}))
    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped)}}
	s.ErrorIs(store.Upsert(withNoAccessCtx, {{$name}}), sac.ErrResourceAccessDenied)
    {{- end }}
	{{- if (.Obj.IsDirectlyScoped)}}
	s.ErrorIs(store.Upsert(withNoAccessToClusterCtx, {{ $name }}), sac.ErrResourceAccessDenied)
	s.ErrorIs(store.Upsert(withAccessToDifferentNsCtx, {{ $name }}), sac.ErrResourceAccessDenied)
	s.NoError(store.Upsert(withAccessCtx, {{ $name }}))
	s.NoError(store.Upsert(withAccessToClusterCtx, {{ $name }}))
	s.ErrorIs(store.UpsertMany(withAccessToDifferentNsCtx, []*{{.Type}}{ {{ $name }} }), sac.ErrResourceAccessDenied)
	s.ErrorIs(store.UpsertMany(withNoAccessToClusterCtx, []*{{.Type}}{ {{ $name }} }), sac.ErrResourceAccessDenied)
	s.NoError(store.UpsertMany(withAccessCtx, []*{{.Type}}{ {{ $name }} }))
	s.NoError(store.UpsertMany(withAccessToClusterCtx, []*{{.Type}}{ {{ $name }} }))
	{{- end }}

	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	s.NoError(store.Delete(ctx, {{template "paramList" $}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})

    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) }}
    s.ErrorIs(store.Delete(withNoAccessCtx, {{template "paramList" $}}), sac.ErrResourceAccessDenied)
    {{- end }}

	var {{$name}}s []*{{.Type}}
    for i := 0; i < 200; i++ {
        {{$name}} := &{{.Type}}{}
        s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
        {{$name}}s = append({{.TrimmedType|lowerCamelCase}}s, {{.TrimmedType|lowerCamelCase}})
    }

	{{- if (.Obj.IsDirectlyScoped)}}
	s.ErrorIs(store.UpsertMany(withAccessToDifferentNsCtx, {{.TrimmedType|lowerCamelCase}}s), sac.ErrResourceAccessDenied)
	{{- end }}
	s.NoError(store.UpsertMany(ctx, {{.TrimmedType|lowerCamelCase}}s))

    {{.TrimmedType|lowerCamelCase}}Count, err = store.Count(ctx)
    s.NoError(err)
    s.Equal({{.TrimmedType|lowerCamelCase}}Count, 200)
    {{- end }}
}

