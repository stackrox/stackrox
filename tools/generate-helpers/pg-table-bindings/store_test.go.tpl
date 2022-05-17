
{{define "paramList"}}{{$name := .TrimmedType|lowerCamelCase}}{{range $idx, $pk := .Schema.PrimaryKeys}}{{if $idx}}, {{end}}{{$pk.Getter $name}}{{end}}{{end}}

{{- $ := . }}
{{- $name := .TrimmedType|lowerCamelCase }}

{{- $namePrefix := .Table|upperCamelCase}}

//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
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
	if s.pool != nil {
		s.pool.Close()
	}
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

    {{if and (not .JoinTable) (eq (len .Schema.RelationshipsToDefineAsForeignKeys) 0) -}}
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

{{- if .GetAll }}
	all{{.TrimmedType|upperCamelCase}}, err := store.GetAll(ctx)
	s.NoError(err)
	s.ElementsMatch({{$name}}s, all{{.TrimmedType|upperCamelCase}})
{{- end }}

    {{.TrimmedType|lowerCamelCase}}Count, err = store.Count(ctx)
    s.NoError(err)
    s.Equal(200, {{.TrimmedType|lowerCamelCase}}Count)
    {{- end }}
}

{{- if eq (len (.Schema.RelationshipsToDefineAsForeignKeys)) 0 }}
{{- if .Obj.IsDirectlyScoped }}

func (s *{{$namePrefix}}StoreSuite) TestSACUpsert() {
	obj := &{{.Type}}{}
	s.NoError(testutils.FullInit(obj, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	ctxs := getSACContexts(obj, storage.Access_READ_WRITE_ACCESS)
	for name, expectedErr := range map[string]error{
		withAllAccess: nil,
		withNoAccess: sac.ErrResourceAccessDenied,
		withNoAccessToCluster: sac.ErrResourceAccessDenied,
		withAccessToDifferentNs: sac.ErrResourceAccessDenied,
		withAccess: nil,
		withAccessToCluster: nil,
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			assert.ErrorIs(t, s.store.Upsert(ctxs[name], obj), expectedErr)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACUpsertMany() {
	obj := &{{.Type}}{}
	s.NoError(testutils.FullInit(obj, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	ctxs := getSACContexts(obj, storage.Access_READ_WRITE_ACCESS)
	for name, expectedErr := range map[string]error{
		withAllAccess: nil,
		withNoAccess: sac.ErrResourceAccessDenied,
		withNoAccessToCluster: sac.ErrResourceAccessDenied,
		withAccessToDifferentNs: sac.ErrResourceAccessDenied,
		withAccess: nil,
		withAccessToCluster: nil,
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			assert.ErrorIs(t, s.store.UpsertMany(ctxs[name], []*{{.Type}}{obj}), expectedErr)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACCount() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)
	s.store.Upsert(withAllAccessCtx, objB)

	ctxs := getSACContexts(objA, storage.Access_READ_ACCESS)
	for name, expectedCount := range map[string]int{
		withAllAccess:           2,
		withNoAccess:            0,
		withNoAccessToCluster:   0,
		withAccessToDifferentNs: 0,
		withAccess:              1,
		withAccessToCluster:     1,
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			count, err := s.store.Count(ctxs[name])
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}


func (s *{{$namePrefix}}StoreSuite) TestSACGetIDs() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)
	s.store.Upsert(withAllAccessCtx, objB)

	ctxs := getSACContexts(objA, storage.Access_READ_ACCESS)
	for name, expectedIds := range map[string][]string{
		withAllAccess:           []string{objA.GetId(), objB.GetId()},
		withNoAccess:            []string{},
		withNoAccessToCluster:   []string{},
		withAccessToDifferentNs: []string{},
		withAccess:              []string{objA.GetId()},
		withAccessToCluster:     []string{objA.GetId()},
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			ids, err := s.store.GetIDs(ctxs[name])
			assert.NoError(t, err)
			assert.EqualValues(t, expectedIds, ids)
		})
	}
}

{{/* TODO(ROX-10624): Remove this condition after all PKs fields were search tagged (PR #1653) */}}
{{- if eq (len .Schema.PrimaryKeys) 1 }}
func (s *{{$namePrefix}}StoreSuite) TestSACExists() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)
	s.store.Upsert(withAllAccessCtx, objB)

	ctxs := getSACContexts(objA, storage.Access_READ_ACCESS)
	for name, expected := range map[string]bool{
		withAllAccess:           true,
		withNoAccess:            false,
		withNoAccessToCluster:   false,
		withAccessToDifferentNs: false,
		withAccess:              true,
		withAccessToCluster:     true,
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			exists, err := s.store.Exists(ctxs[name], objA.GetId())
			assert.NoError(t, err)
			assert.Equal(t, expected, exists)
		})
	}
}
{{- end }}

const (
	withAllAccess = "AllAccess"
	withNoAccess = "NoAccess"
	withAccessToDifferentNs = "AccessToDifferentNs"
	withAccess = "Access"
	withAccessToCluster = "AccessToCluster"
	withNoAccessToCluster = "NoAccessToCluster"
)

func getSACContexts(obj *{{.Type}}, access storage.Access) map[string]context.Context {
	return map[string]context.Context {
		withAllAccess: sac.WithAllAccess(context.Background()),
		withNoAccess: sac.WithNoAccess(context.Background()),
		withAccessToDifferentNs: sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(access),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys({{ "obj" | .Obj.GetClusterID }}),
				sac.NamespaceScopeKeys("unknown ns"),
		)),
		withAccess: sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(access),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys({{ "obj" | .Obj.GetClusterID }}),
				{{- if .Obj.IsNamespaceScope }}
					sac.NamespaceScopeKeys({{ "obj" | .Obj.GetNamespace }}),
				{{- end }}
		)),
		withAccessToCluster: sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(access),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys({{ "obj" | .Obj.GetClusterID }}),
		)),
		withNoAccessToCluster: sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(access),
				sac.ResourceScopeKeys(targetResource),
				sac.ClusterScopeKeys("unknown cluster"),
		)),
	}
}
{{ end }}
{{- end }}
