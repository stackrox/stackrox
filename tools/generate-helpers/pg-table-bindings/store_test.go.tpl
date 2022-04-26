
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
	store Store
	pool *pgxpool.Pool
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

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.Require().NoError(err)

	Destroy(ctx, pool)

	s.pool = pool
	s.store = New(ctx, pool)
}

func (s *{{$namePrefix}}StoreSuite) TearDownTest() {
	s.pool.Close()
	s.envIsolator.RestoreAll()
}

func (s *{{$namePrefix}}StoreSuite) TestStore() {
    ctx := sac.WithAllAccess(context.Background())

	store := s.store

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

	s.NoError(store.Upsert(ctx, {{$name}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	{{$name}}Count, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(1, {{$name}}Count)

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

	s.NoError(store.UpsertMany(ctx, {{.TrimmedType|lowerCamelCase}}s))

    {{.TrimmedType|lowerCamelCase}}Count, err = store.Count(ctx)
    s.NoError(err)
    s.Equal(200, {{.TrimmedType|lowerCamelCase}}Count)
    {{- end }}
}

{{- if .Obj.IsDirectlyScoped }}
func (s *{{$namePrefix}}StoreSuite) TestSAC() {
	obj := &{{.Type}}{}
	s.NoError(testutils.FullInit(obj, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	withNoAccessCtx := sac.WithNoAccess(context.Background())
	withAccessToDifferentNsCtx:= sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(targetResource),
			sac.ClusterScopeKeys({{ "obj" | .Obj.GetClusterID }}),
			sac.NamespaceScopeKeys("unknown ns"),
	))
	withAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(targetResource),
			sac.ClusterScopeKeys({{ "obj" | .Obj.GetClusterID }}),
			{{- if .Obj.IsNamespaceScope }}
			sac.NamespaceScopeKeys({{ "obj" | .Obj.GetNamespace }}),
			{{- end }}
	))
	withAccessToClusterCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys({{ "obj" | .Obj.GetClusterID }}),
	))
	withNoAccessToClusterCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys("unknown cluster"),
	))

	store := s.store

	for ctx, expectedErr := range map[context.Context]error{
		withAllAccessCtx: nil,
		withNoAccessCtx: sac.ErrResourceAccessDenied,
		withNoAccessToClusterCtx: sac.ErrResourceAccessDenied,
		withAccessToDifferentNsCtx: sac.ErrResourceAccessDenied,
		withAccessCtx: nil,
		withAccessToClusterCtx: nil,
	} {
		s.T().Run("Upsert", func(t *testing.T) {
			assert.ErrorIs(t, store.Upsert(ctx, obj), expectedErr)
		})
		s.T().Run("UpsertMany", func(t *testing.T) {
			assert.ErrorIs(t, store.UpsertMany(ctx, []*{{.Type}}{obj}), expectedErr)
		})
	}
}
{{ end }}