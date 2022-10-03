
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
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type {{$namePrefix}}StoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	store Store
	testDB *pgtest.TestPostgres
}

func Test{{$namePrefix}}Store(t *testing.T) {
	suite.Run(t, new({{$namePrefix}}StoreSuite))
}

func (s *{{$namePrefix}}StoreSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.Pool)
}

func (s *{{$namePrefix}}StoreSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	tag, err := s.testDB.Exec(ctx, "TRUNCATE {{ .Schema.Table }} CASCADE")
	s.T().Log("{{ .Schema.Table }}", tag)
	s.NoError(err)
}

func (s *{{$namePrefix}}StoreSuite) TearDownSuite() {
    s.testDB.Teardown(s.T())
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
    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
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

    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{$name}}Count, err = store.Count(withNoAccessCtx)
    s.NoError(err)
    s.Zero({{$name}}Count)
    {{- end }}

	{{$name}}Exists, err := store.Exists(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True({{$name}}Exists)
	s.NoError(store.Upsert(ctx, {{$name}}))
    {{- if or (.Obj.IsGloballyScoped) (.Obj.HasPermissionChecker) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
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
	{{- else }}
	s.NoError(store.Delete(withNoAccessCtx, {{template "paramList" $}}))
	{{- end }}

	var {{$name}}s []*{{.Type}}
    for i := 0; i < 200; i++ {
        {{$name}} := &{{.Type}}{}
        s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
        {{- if .Cycle}}
        {{$name}}.{{.EmbeddedFK}} = {{$name}}.{{.ReferencedField}}
        {{- end}}
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

func (s *{{$namePrefix}}StoreSuite) TestSACWalk() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)
	s.store.Upsert(withAllAccessCtx, objB)

	ctxs := getSACContexts(objA, storage.Access_READ_ACCESS)
	for name, expectedIds := range map[string][]string{
		withAllAccess:           []string{ {{ (index .Schema.PrimaryKeys 0).Getter "objA"}}, {{ (index .Schema.PrimaryKeys 0).Getter "objB"}} },
		withNoAccess:            []string{},
		withNoAccessToCluster:   []string{},
		withAccessToDifferentNs: []string{},
		withAccess:              []string{ {{ (index .Schema.PrimaryKeys 0).Getter "objA"}} },
		withAccessToCluster:     []string{ {{ (index .Schema.PrimaryKeys 0).Getter "objA"}} },
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			ids := []string{}
			getIds := func(obj *{{.Type}}) error {
				ids = append(ids,  {{ (index .Schema.PrimaryKeys 0).Getter "obj" }} )
				return nil
			}
			err := s.store.Walk(ctxs[name], getIds)
			assert.NoError(t, err)
			assert.ElementsMatch(t, expectedIds, ids)
		})
	}
}

{{ if eq (len .Schema.PrimaryKeys) 1 }}
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
		withAllAccess:           []string{ {{ "objA" | .Obj.GetID }}, {{ "objB" | .Obj.GetID }} },
		withNoAccess:            []string{},
		withNoAccessToCluster:   []string{},
		withAccessToDifferentNs: []string{},
		withAccess:              []string{ {{ "objA" | .Obj.GetID }} },
		withAccessToCluster:     []string{ {{ "objA" | .Obj.GetID }} },
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			ids, err := s.store.GetIDs(ctxs[name])
			assert.NoError(t, err)
			assert.EqualValues(t, expectedIds, ids)
		})
	}
}
{{ end }}

func (s *{{$namePrefix}}StoreSuite) TestSACExists() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)

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
			exists, err := s.store.Exists(ctxs[name], {{ range $field := .Schema.PrimaryKeys }}{{$field.Getter "objA"}}, {{end}})
			assert.NoError(t, err)
			assert.Equal(t, expected, exists)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACGet() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)

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
			actual, exists, err := s.store.Get(ctxs[name], {{ range $field := .Schema.PrimaryKeys }}{{$field.Getter "objA"}}, {{end}})
			assert.NoError(t, err)
			assert.Equal(t, expected, exists)
			if expected == true {
				assert.Equal(t, objA, actual)
			} else {
				assert.Nil(t, actual)
			}
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACDelete() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	withAllAccessCtx := sac.WithAllAccess(context.Background())

	ctxs := getSACContexts(objA, storage.Access_READ_WRITE_ACCESS)
	for name, expectedCount := range map[string]int{
		withAllAccess:           0,
		withNoAccess:            2,
		withNoAccessToCluster:   2,
		withAccessToDifferentNs: 2,
		withAccess:              1,
		withAccessToCluster:     1,
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			s.SetupTest()

			s.NoError(s.store.Upsert(withAllAccessCtx, objA))
			s.NoError(s.store.Upsert(withAllAccessCtx, objB))

			assert.NoError(t, s.store.Delete(ctxs[name], {{ range $field := .Schema.PrimaryKeys }}{{$field.Getter "objA"}}, {{end}}))
			assert.NoError(t, s.store.Delete(ctxs[name], {{ range $field := .Schema.PrimaryKeys }}{{$field.Getter "objB"}}, {{end}}))

			count, err := s.store.Count(withAllAccessCtx)
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

{{ if eq (len .Schema.PrimaryKeys) 1 }}
func (s *{{$namePrefix}}StoreSuite) TestSACDeleteMany() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	withAllAccessCtx := sac.WithAllAccess(context.Background())

	ctxs := getSACContexts(objA, storage.Access_READ_WRITE_ACCESS)
	for name, expectedCount := range map[string]int{
		withAllAccess:           0,
		withNoAccess:            2,
		withNoAccessToCluster:   2,
		withAccessToDifferentNs: 2,
		withAccess:              1,
		withAccessToCluster:     1,
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			s.SetupTest()

			s.NoError(s.store.Upsert(withAllAccessCtx, objA))
			s.NoError(s.store.Upsert(withAllAccessCtx, objB))

			assert.NoError(t, s.store.DeleteMany(ctxs[name], []string{
				{{ (index .Schema.PrimaryKeys 0).Getter "objA"}},
				{{ (index .Schema.PrimaryKeys 0).Getter "objB"}},
			}))

			count, err := s.store.Count(withAllAccessCtx)
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACGetMany() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.store.Upsert(withAllAccessCtx, objA)
	s.store.Upsert(withAllAccessCtx, objB)

	ctxs := getSACContexts(objA, storage.Access_READ_ACCESS)
	for name, expected := range map[string]struct{
		elems []*{{ .Type }}
		missingIndices []int
	}{
		withAllAccess:           { elems: []*{{ .Type }}{objA, objB}, missingIndices: []int{}},
		withNoAccess:            { elems: []*{{ .Type }}{}, missingIndices: []int{0, 1}},
		withNoAccessToCluster:   { elems: []*{{ .Type }}{}, missingIndices: []int{0, 1}},
		withAccessToDifferentNs: { elems: []*{{ .Type }}{}, missingIndices: []int{0, 1}},
		withAccess:              { elems: []*{{ .Type }}{objA}, missingIndices: []int{1}},
		withAccessToCluster:     { elems: []*{{ .Type }}{objA}, missingIndices: []int{1}},
	} {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			actual, missingIndices, err := s.store.GetMany(ctxs[name], []string{ {{ "objA" | .Obj.GetID }}, {{ "objB" | .Obj.GetID }} })
			assert.NoError(t, err)
			assert.Equal(t, expected.elems, actual)
			assert.Equal(t, expected.missingIndices, missingIndices)
		})
	}

	s.T().Run("with no ids", func(t *testing.T) {
		actual, missingIndices, err := s.store.GetMany(withAllAccessCtx, []string{})
		assert.Nil(t, err)
		assert.Nil(t, actual)
		assert.Nil(t, missingIndices)
	})
}
{{ end }}

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
{{end}}
{{- end }}
