{{- $namePrefix := .Table|upperCamelCase}}

// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
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
	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		panic(err)
	}
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	s.NoError(err)
	defer pool.Close()

	Destroy(pool)
	store := New(pool)

	{{.TrimmedType|lowerCamelCase}} := fixtures.Get{{.TrimmedType}}()
	found{{.TrimmedType|upperCamelCase}}, exists, err := store.Get({{.TrimmedType|lowerCamelCase}}.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})

	s.NoError(store.Upsert({{.TrimmedType|lowerCamelCase}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get({{.TrimmedType|lowerCamelCase}}.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal({{.TrimmedType|lowerCamelCase}}, found{{.TrimmedType|upperCamelCase}})

	{{.TrimmedType|lowerCamelCase}}Count, err := store.Count()
	s.NoError(err)
	s.Equal({{.TrimmedType|lowerCamelCase}}Count, 1)

	{{.TrimmedType|lowerCamelCase}}Exists, err := store.Exists({{.TrimmedType|lowerCamelCase}}.GetId())
	s.NoError(err)
	s.True({{.TrimmedType|lowerCamelCase}}Exists)
	s.NoError(store.Upsert({{.TrimmedType|lowerCamelCase}}))

	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get({{.TrimmedType|lowerCamelCase}}.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal({{.TrimmedType|lowerCamelCase}}, found{{.TrimmedType|upperCamelCase}})

	s.NoError(store.Delete({{.TrimmedType|lowerCamelCase}}.GetId()))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get({{.TrimmedType|lowerCamelCase}}.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})
}

