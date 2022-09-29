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
	"github.com/stackrox/rox/pkg/uuid"
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

func (s *{{$namePrefix}}StoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.Pool)
}

func (s *{{$namePrefix}}StoreSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
	s.envIsolator.RestoreAll()
}

func (s *{{$namePrefix}}StoreSuite) TestStore() {
    ctx := sac.WithAllAccess(context.Background())

	store := s.store

	{{$name}} := &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	found{{.TrimmedType|upperCamelCase}}, exists, err := store.Get(ctx)
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})

    withNoAccessCtx := sac.WithNoAccess(ctx)

	s.NoError(store.Upsert(ctx, {{$name}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	s.NoError(store.Delete(ctx))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx)
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})

    s.ErrorIs(store.Delete(withNoAccessCtx), sac.ErrResourceAccessDenied)

	{{$name}} = &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	s.NoError(store.Upsert(ctx, {{$name}}))

	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	{{$name}} = &{{.Type}}{}
	s.NoError(testutils.FullInit({{$name}}, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	s.NoError(store.Upsert(ctx, {{$name}}))

	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})
}
