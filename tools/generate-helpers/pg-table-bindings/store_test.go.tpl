
{{define "paramList"}}{{$name := .TrimmedType|lowerCamelCase}}{{range $index, $pk := .Schema.PrimaryKeys}}{{if $index}}, {{end}}{{$pk.Getter $name}}{{end}}{{end}}

{{- $ := . }}
{{- $name := .TrimmedType|lowerCamelCase }}

{{- $namePrefix := .Table|upperCamelCase}}

//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/stackrox/rox/pkg/uuid"
)

type {{$namePrefix}}StoreSuite struct {
	suite.Suite
	store Store
	testDB *pgtest.TestPostgres
}

func Test{{$namePrefix}}Store(t *testing.T) {
	suite.Run(t, new({{$namePrefix}}StoreSuite))
}

func (s *{{$namePrefix}}StoreSuite) SetupSuite() {
	{{ if .FeatureFlag }}
	s.T().Setenv(features.{{.FeatureFlag}}.EnvVar(), "true")
	if !features.{{.FeatureFlag}}.Enabled() {
		s.T().Skip("Skip postgres store tests because feature flag is off")
		s.T().SkipNow()
	}
	{{- end }}

	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
}

func (s *{{$namePrefix}}StoreSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	tag, err := s.testDB.Exec(ctx, "TRUNCATE {{ .Schema.Table }} CASCADE")
	s.T().Log("{{ .Schema.Table }}", tag)
	s.NoError(err)
}

func (s *{{$namePrefix}}StoreSuite) TearDownSuite() {
    s.testDB.Teardown(s.T())
}

func (s *{{$namePrefix}}StoreSuite) TestStore() {
    ctx := sac.WithAllAccess(context.Background())

	store := s.store

	{{$name}} := &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	{{- if .Cycle}}
	{{$name}}.{{.EmbeddedFK}} = nil
	{{- end}}

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
	var {{$name}}IDs []string
    for i := 0; i < 200; i++ {
        {{$name}} := &{{.Type}}{}
        s.NoError(testutils.FullInit({{$name}}, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
        {{- if .Cycle}}
        {{$name}}.{{.EmbeddedFK}} = nil
        {{- end}}
        {{$name}}s = append({{.TrimmedType|lowerCamelCase}}s, {{.TrimmedType|lowerCamelCase}})
        {{$name}}IDs = append({{$name}}IDs, {{template "paramList" $}})
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

    s.NoError(store.DeleteMany(ctx, {{$name}}IDs))

    {{.TrimmedType|lowerCamelCase}}Count, err = store.Count(ctx)
    s.NoError(err)
    s.Equal(0, {{.TrimmedType|lowerCamelCase}}Count)
    {{- end }}
}

{{- if eq (len (.Schema.RelationshipsToDefineAsForeignKeys)) 0 }}
{{- if .Obj.IsDirectlyScoped }}
const (
	withAllAccess = "AllAccess"
	withNoAccess = "NoAccess"
	withAccessToDifferentNs = "AccessToDifferentNs"
	withAccess = "Access"
	withAccessToCluster = "AccessToCluster"
	withNoAccessToCluster = "NoAccessToCluster"
)

var (
	withAllAccessCtx = sac.WithAllAccess(context.Background())
)

type testCase struct {
	context                context.Context
	expectedObjIDs         []string
	expectedIdentifiers    []string
	expectedMissingIndices []int
	expectedObjects        []*{{.Type}}
	expectedWriteError     error
}
func (s *{{$namePrefix}}StoreSuite) getTestData(access storage.Access) (*{{.Type}}, *{{.Type}}, map[string]testCase) {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	objB := &{{.Type}}{}
	s.NoError(testutils.FullInit(objB, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	testCases := map[string]testCase{
		withAllAccess: {
			context:                sac.WithAllAccess(context.Background()),
			expectedObjIDs:         []string{ {{ "objA" | .Obj.GetID }}, {{ "objB" | .Obj.GetID }} },
			expectedIdentifiers:    []string{ {{ (index .Schema.PrimaryKeys 0).Getter "objA" }}, {{ (index .Schema.PrimaryKeys 0).Getter "objB" }} },
			expectedMissingIndices: []int{},
			expectedObjects:        []*{{.Type}}{objA, objB},
			expectedWriteError:     nil,
		},
		withNoAccess: {
			context:                sac.WithNoAccess(context.Background()),
			expectedObjIDs:         []string{},
			expectedIdentifiers:    []string{},
			expectedMissingIndices: []int{0, 1},
			expectedObjects:        []*{{.Type}}{},
			expectedWriteError:     sac.ErrResourceAccessDenied,
		},
		withNoAccessToCluster: {
			context:                sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(access),
					sac.ResourceScopeKeys(targetResource),
					sac.ClusterScopeKeys(uuid.Nil.String()),
				),
			),
			expectedObjIDs:         []string{},
			expectedIdentifiers:    []string{},
			expectedMissingIndices: []int{0, 1},
			expectedObjects:        []*{{.Type}}{},
			expectedWriteError:     sac.ErrResourceAccessDenied,
		},
		withAccessToDifferentNs: {
			context:                sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(access),
					sac.ResourceScopeKeys(targetResource),
					sac.ClusterScopeKeys({{ "objA" | .Obj.GetClusterID }}),
					sac.NamespaceScopeKeys("unknown ns"),
				),
			),
			expectedObjIDs:         []string{},
			expectedIdentifiers:    []string{},
			expectedMissingIndices: []int{0, 1},
			expectedObjects:        []*{{.Type}}{},
			expectedWriteError:     sac.ErrResourceAccessDenied,
		},
		withAccess: {
			context:                sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(access),
					sac.ResourceScopeKeys(targetResource),
					sac.ClusterScopeKeys({{ "objA" | .Obj.GetClusterID }}),
					{{- if .Obj.IsNamespaceScope }}
					sac.NamespaceScopeKeys({{ "objA" | .Obj.GetNamespace }}),
					{{- end }}
				),
			),
			expectedObjIDs:         []string{ {{ "objA" | .Obj.GetID }} },
			expectedIdentifiers:    []string{ {{ (index .Schema.PrimaryKeys 0).Getter "objA" }} },
			expectedMissingIndices: []int{1},
			expectedObjects:        []*{{.Type}}{objA},
			expectedWriteError:     nil,
		},
		withAccessToCluster: {
			context:                sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(access),
					sac.ResourceScopeKeys(targetResource),
					sac.ClusterScopeKeys({{ "objA" | .Obj.GetClusterID }}),
				),
			),
			expectedObjIDs:         []string{ {{ "objA" | .Obj.GetID }} },
			expectedIdentifiers:    []string{ {{ (index .Schema.PrimaryKeys 0).Getter "objA" }} },
			expectedMissingIndices: []int{1},
			expectedObjects:        []*{{.Type}}{objA},
			expectedWriteError:     nil,
		},
	}

	return objA, objB, testCases
}

func (s *{{$namePrefix}}StoreSuite) TestSACUpsert() {
	obj, _, testCases := s.getTestData(storage.Access_READ_WRITE_ACCESS)
	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			assert.ErrorIs(t, s.store.Upsert(testCase.context, obj), testCase.expectedWriteError)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACUpsertMany() {
	obj, _, testCases := s.getTestData(storage.Access_READ_WRITE_ACCESS)
	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			assert.ErrorIs(t, s.store.UpsertMany(testCase.context, []*{{.Type}}{obj}), testCase.expectedWriteError)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACCount() {
	objA, objB, testCases := s.getTestData(storage.Access_READ_ACCESS)
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objA))
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objB))

	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			expectedCount := len(testCase.expectedObjects)
			count, err := s.store.Count(testCase.context)
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACWalk() {
	objA, objB, testCases := s.getTestData(storage.Access_READ_ACCESS)
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objA))
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objB))

	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			identifiers := []string{}
			getIDs := func(obj *{{.Type}}) error {
				identifiers = append(identifiers, {{ (index .Schema.PrimaryKeys 0).Getter "obj" }} )
				return nil
			}
			err := s.store.Walk(testCase.context, getIDs)
			assert.NoError(t, err)
			assert.ElementsMatch(t, testCase.expectedIdentifiers, identifiers)
		})
	}
}

{{ if eq (len .Schema.PrimaryKeys) 1 }}
func (s *{{$namePrefix}}StoreSuite) TestSACGetIDs() {
	objA, objB, testCases := s.getTestData(storage.Access_READ_ACCESS)
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objA))
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objB))

	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			identifiers, err := s.store.GetIDs(testCase.context)
			assert.NoError(t, err)
			assert.EqualValues(t, testCase.expectedObjIDs, identifiers)
		})
	}
}
{{ end }}

func (s *{{$namePrefix}}StoreSuite) TestSACExists() {
	objA, _, testCases := s.getTestData(storage.Access_READ_ACCESS)
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objA))

	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			// Assumption from the test case structure: objA is always in the visible list
			// in the first position.
			expectedFound := len(testCase.expectedObjects) > 0
			exists, err := s.store.Exists(testCase.context, {{ range $field := .Schema.PrimaryKeys }}{{$field.Getter "objA"}}, {{end}})
			assert.NoError(t, err)
			assert.Equal(t, expectedFound, exists)
		})
	}
}

func (s *{{$namePrefix}}StoreSuite) TestSACGet() {
	objA := &{{.Type}}{}
	s.NoError(testutils.FullInit(objA, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	withAllAccessCtx := sac.WithAllAccess(context.Background())
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objA))

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
	objA, objB, testCases := s.getTestData(storage.Access_READ_ACCESS)
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objA))
	s.Require().NoError(s.store.Upsert(withAllAccessCtx, objB))

	for name, testCase := range testCases {
		s.T().Run(fmt.Sprintf("with %s", name), func(t *testing.T) {
			actual, missingIndices, err := s.store.GetMany(testCase.context, []string{ {{ "objA" | .Obj.GetID }}, {{ "objB" | .Obj.GetID }} })
			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedObjects, actual)
			assert.Equal(t, testCase.expectedMissingIndices, missingIndices)
		})
	}

	s.T().Run("with no identifiers", func(t *testing.T) {
		actual, missingIndices, err := s.store.GetMany(withAllAccessCtx, []string{})
		assert.Nil(t, err)
		assert.Nil(t, actual)
		assert.Nil(t, missingIndices)
	})
}
{{ end }}

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
				sac.ClusterScopeKeys(uuid.Nil.String()),
		)),
	}
}
{{end}}
{{- end }}
