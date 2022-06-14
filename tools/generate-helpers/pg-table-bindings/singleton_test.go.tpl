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
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/postgres/pgtest"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stackrox/stackrox/pkg/uuid"
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
