
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
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils"
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
    ctx := context.Background()

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

	s.NoError(store.Upsert(ctx, {{$name}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	{{$name}}Count, err := store.Count(ctx)
	s.NoError(err)
	s.Equal({{$name}}Count, 1)

	{{$name}}Exists, err := store.Exists(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True({{$name}}Exists)
	s.NoError(store.Upsert(ctx, {{$name}}))

	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.True(exists)
	s.Equal({{$name}}, found{{.TrimmedType|upperCamelCase}})

	s.NoError(store.Delete(ctx, {{template "paramList" $}}))
	found{{.TrimmedType|upperCamelCase}}, exists, err = store.Get(ctx, {{template "paramList" $}})
	s.NoError(err)
	s.False(exists)
	s.Nil(found{{.TrimmedType|upperCamelCase}})
}

