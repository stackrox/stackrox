{{define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $idx, $pk := .}}{{if $idx}} AND {{end}}{{$pk.ColumnName}} = ${{add $idx 1}}{{end}}{{end}}
{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}

{{- $ := . }}
{{- $pks := .Schema.PrimaryKeys }}

{{- $singlePK := false }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{/*If there are multiple pks, then use the explicitly specified id column.*/}}
{{- else if .Schema.ID.ColumnName}}
{{ $singlePK = .Schema.ID }}
{{- end }}
{{ $inMigration := ne (index . "Migration") nil}}

package postgres

import (
    "context"
    "strings"
    "time"

    "github.com/gogo/protobuf/proto"
    "github.com/hashicorp/go-multierror"
    "github.com/jackc/pgx/v4"
    "github.com/jackc/pgx/v4/pgxpool"
    "github.com/pkg/errors"
    {{- if not $inMigration}}
    "github.com/stackrox/rox/central/metrics"
    "github.com/stackrox/rox/central/role/resources"
    {{- end}}
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/auth/permissions"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
    "github.com/stackrox/rox/pkg/sac"
    "github.com/stackrox/rox/pkg/search"
    "github.com/stackrox/rox/pkg/search/postgres"
    "github.com/stackrox/rox/pkg/sync"
    "gorm.io/gorm"
)

const (
        baseTable = "{{.Table}}"

        batchAfter = 100

        // using copyFrom, we may not even want to batch.  It would probably be simpler
        // to deal with failures if we just sent it all.  Something to think about as we
        // proceed and move into more e2e and larger performance testing
        batchSize = 10000

        cursorBatchSize = 50
)

var (
    log = logging.LoggerForModule()
    schema = {{ template "schemaVar" .Schema}}
    {{- if or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
        {{- if not $inMigration}}
        targetResource = resources.{{.Type | storageToResource}}
        {{- end}}
    {{- end }}
)

type Store interface {
    Count(ctx context.Context) (int, error)
    Exists(ctx context.Context, {{template "paramList" $pks}}) (bool, error)
    Get(ctx context.Context, {{template "paramList" $pks}}) (*{{.Type}}, bool, error)
{{- if .SearchCategory }}
    GetByQuery(ctx context.Context, query *v1.Query) ([]*{{.Type}}, error)
{{- end }}
{{- if .GetAll }}
    GetAll(ctx context.Context) ([]*{{.Type}}, error)
{{- end }}
{{- if not .JoinTable }}
    Upsert(ctx context.Context, obj *{{.Type}}) error
    UpsertMany(ctx context.Context, objs []*{{.Type}}) error
    Delete(ctx context.Context, {{template "paramList" $pks}}) error
    DeleteByQuery(ctx context.Context, q *v1.Query) error
{{- end }}

{{- if $singlePK }}
    GetIDs(ctx context.Context) ([]{{$singlePK.Type}}, error)
    GetMany(ctx context.Context, ids []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error)
{{- if not .JoinTable }}
    DeleteMany(ctx context.Context, ids []{{$singlePK.Type}}) error
{{- end }}
{{- end }}

    Walk(ctx context.Context, fn func(obj *{{.Type}}) error) error

    AckKeysIndexed(ctx context.Context, keys ...string) error
    GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
    db *pgxpool.Pool
    mutex sync.Mutex
}

{{ define "defineScopeChecker" }}scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_{{ . }}_ACCESS).Resource(targetResource){{ end }}

{{define "createTableStmtVar"}}pkgSchema.CreateTable{{.Table|upperCamelCase}}Stmt{{end}}

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool) Store {
    return &storeImpl{
        db: db,
    }
}

{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := .schema }}
func {{ template "insertFunctionName" $schema }}(ctx context.Context, batch *pgx.Batch, obj {{$schema.Type}}{{ range $field := $schema.FieldsDeterminedByParent }}, {{$field.Name}} {{$field.Type}}{{end}}) error {
    {{if not $schema.Parent }}
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }
    {{end}}

    values := []interface{} {
        // parent primary keys start
        {{- range $field := $schema.DBColumnFields -}}
        {{- if eq $field.DataType "datetime" }}
        pgutils.NilOrTime({{$field.Getter "obj"}}),
        {{- else }}
        {{$field.Getter "obj"}},{{end}}
        {{- end}}
    }

    finalStr := "INSERT INTO {{$schema.Table}} ({{template "commaSeparatedColumns" $schema.DBColumnFields }}) VALUES({{ valueExpansion (len $schema.DBColumnFields) }}) ON CONFLICT({{template "commaSeparatedColumns" $schema.PrimaryKeys}}) DO UPDATE SET {{template "updateExclusions" $schema.DBColumnFields}}"
    batch.Queue(finalStr, values...)

    {{ if $schema.Children }}
    var query string
    {{end}}

    {{range $idx, $child := $schema.Children }}
    for childIdx, child := range obj.{{$child.ObjectGetter}} {
        if err := {{ template "insertFunctionName" $child }}(ctx, batch, child{{ range $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, childIdx); err != nil {
            return err
        }
    }

    query = "delete from {{$child.Table}} where {{ range $idx, $field := $child.FieldsReferringToParent }}{{if $idx}} AND {{end}}{{$field.ColumnName}} = ${{add $idx 1}}{{end}} AND idx >= ${{add (len $child.FieldsReferringToParent) 1}}"
    batch.Queue(query{{ range $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, len(obj.{{$child.ObjectGetter}}))
    {{- end}}
    return nil
}

{{range $idx, $child := $schema.Children}}{{ template "insertObject" dict "schema" $child "joinTable" false }}{{end}}
{{- end}}

{{- if not .JoinTable }}
{{ template "insertObject" dict "schema" .Schema "joinTable" .JoinTable }}
{{- end}}

{{- define "copyFunctionName"}}{{- $schema := . }}copyFrom{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "copyObject"}}
{{- $schema := . }}
func (s *storeImpl) {{ template "copyFunctionName" $schema }}(ctx context.Context, tx pgx.Tx, {{ range $idx, $field := $schema.FieldsReferringToParent }} {{$field.Name}} {{$field.Type}},{{end}} objs ...{{$schema.Type}}) error {

    inputRows := [][]interface{}{}

    var err error

    {{if and (eq (len $schema.PrimaryKeys) 1) (not $schema.Parent) }}
    // This is a copy so first we must delete the rows and re-add them
    // Which is essentially the desired behaviour of an upsert.
    var deletes []string
    {{end}}

    copyCols := []string {
    {{range $idx, $field := $schema.DBColumnFields}}
        "{{$field.ColumnName|lowerCase}}",
    {{end}}
    }

    for idx, obj := range objs {
        // Todo: ROX-9499 Figure out how to more cleanly template around this issue.
        log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj in the loop is not used as it only consists of the parent id and the idx.  Putting this here as a stop gap to simply use the object.  %s", obj)

        {{/* If embedded, the top-level has the full serialized object */}}
        {{if not $schema.Parent }}
        serialized, marshalErr := obj.Marshal()
        if marshalErr != nil {
            return marshalErr
        }
        {{end}}

        inputRows = append(inputRows, []interface{}{
            {{ range $idx, $field := $schema.DBColumnFields }}
            {{if eq $field.DataType "datetime"}}
            pgutils.NilOrTime({{$field.Getter "obj"}}),
            {{- else}}
            {{$field.Getter "obj"}},{{end}}
            {{end}}
        })

        {{ if not $schema.Parent }}
        {{if eq (len $schema.PrimaryKeys) 1}}
        // Add the id to be deleted.
        deletes = append(deletes, {{ range $field := $schema.PrimaryKeys }}{{$field.Getter "obj"}}, {{end}})
        {{else}}
        if err := s.Delete(ctx, {{ range $field := $schema.PrimaryKeys }}{{$field.Getter "obj"}}, {{end}}); err != nil {
            return err
        }

        {{end}}
        {{end}}

        // if we hit our batch size we need to push the data
        if (idx + 1) % batchSize == 0 || idx == len(objs) - 1  {
            // copy does not upsert so have to delete first.  parent deletion cascades so only need to
            // delete for the top level parent
            {{if and ((eq (len $schema.PrimaryKeys) 1)) (not $schema.Parent) }}
            if err := s.DeleteMany(ctx, deletes); err != nil {
                return err
            }
            // clear the inserts and vals for the next batch
            deletes = nil
            {{end}}

            _, err = tx.CopyFrom(ctx, pgx.Identifier{"{{$schema.Table|lowerCase}}"}, copyCols, pgx.CopyFromRows(inputRows))

            if err != nil {
                return err
            }

            // clear the input rows for the next batch
            inputRows = inputRows[:0]
        }
    }

    {{if $schema.Children }}
    for idx, obj := range objs {
        _ = idx // idx may or may not be used depending on how nested we are, so avoid compile-time errors.
        {{range $child := $schema.Children }}
        if err = s.{{ template "copyFunctionName" $child }}(ctx, tx{{ range $idx, $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, obj.{{$child.ObjectGetter}}...); err != nil {
            return err
        }
        {{- end}}
    }
    {{end}}

    return err
}
{{range $child := $schema.Children}}{{ template "copyObject" $child }}{{end}}
{{- end}}

{{- if not .JoinTable }}
{{- if not .NoCopyFrom }}
{{ template "copyObject" .Schema }}
{{- end }}
{{- end }}

{{- if not .JoinTable }}
{{- if not .NoCopyFrom }}
func (s *storeImpl) copyFrom(ctx context.Context, objs ...*{{.Type}}) error {
    conn, release, err := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
	if err != nil {
	    return err
	}
    defer release()

    tx, err := conn.Begin(ctx)
    if err != nil {
        return err
    }

    if err := s.{{ template "copyFunctionName" .Schema }}(ctx, tx, objs...); err != nil {
        if err := tx.Rollback(ctx); err != nil {
            return err
        }
        return err
    }
    if err := tx.Commit(ctx); err != nil {
        return err
    }
    return nil
}
{{- end}}

func (s *storeImpl) upsert(ctx context.Context, objs ...*{{.Type}}) error {
    conn, release, err := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
	if err != nil {
	    return err
	}
	defer release()

    for _, obj := range objs {
        batch := &pgx.Batch{}
	    if err := {{ template "insertFunctionName" .Schema }}(ctx, batch, obj); err != nil {
		    return err
        }
		batchResults := conn.SendBatch(ctx, batch)
		var result *multierror.Error
		for i := 0; i < batch.Len(); i++ {
			_, err := batchResults.Exec()
			result = multierror.Append(result, err)
		}
		if err := batchResults.Close(); err != nil {
			return err
		}
		if err := result.ErrorOrNil(); err != nil {
			return err
		}
    }
    return nil
}

func (s *storeImpl) Upsert(ctx context.Context, obj *{{.Type}}) error {
    {{- if not $inMigration}}
    defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "{{.TrimmedType}}")

    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.UpsertAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsGloballyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    {{- else if and (.Obj.IsDirectlyScoped) (.Obj.IsClusterScope) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}.
        ClusterID({{ "obj" | .Obj.GetClusterID }})
    {{- else if and (.Obj.IsDirectlyScoped) (.Obj.IsNamespaceScope) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}.
        ClusterID({{ "obj" | .Obj.GetClusterID }}).Namespace({{ "obj" | .Obj.GetNamespace }})
    {{- end }}
    {{- if or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped)  }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- end }}
    {{- end }}{{/* if not $inMigration */}}

	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

func (s *storeImpl) UpsertMany(ctx context.Context, objs []*{{.Type}}) error {
    {{- if not $inMigration}}
    defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "{{.TrimmedType}}")

    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.UpsertManyAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsGloballyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- else if .Obj.IsDirectlyScoped -}}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        var deniedIds []string
        for _, obj := range objs {
            {{- if .Obj.IsClusterScope }}
            subScopeChecker := scopeChecker.ClusterID({{ "obj" | .Obj.GetClusterID }})
            {{- else if .Obj.IsNamespaceScope }}
            subScopeChecker := scopeChecker.ClusterID({{ "obj" | .Obj.GetClusterID }}).Namespace({{ "obj" | .Obj.GetNamespace }})
            {{- end }}
            if !subScopeChecker.IsAllowed() {
                deniedIds = append(deniedIds, {{ "obj" | .Obj.GetID }})
            }
        }
        if len(deniedIds) != 0 {
            return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying {{ .TrimmedType|lowerCamelCase }}s with IDs [%s] was denied", strings.Join(deniedIds, ", "))
        }
    }
    {{- end }}
    {{- end }}{{/* if not $inMigration */}}

    {{- if .NoCopyFrom }}
    return s.upsert(ctx, objs...)
    {{- else }}

	return pgutils.Retry(func() error {
		// Lock since copyFrom requires a delete first before being executed.  If multiple processes are updating
		// same subset of rows, both deletes could occur before the copyFrom resulting in unique constraint
		// violations
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if len(objs) < batchAfter {
			return s.upsert(ctx, objs...)
		} else {
			return s.copyFrom(ctx, objs...)
		}
	})
    {{- end }}
}
{{- end }}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    var sacQueryFilter *v1.Query

    {{ if not $inMigration}}
    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.CountAllowed(ctx); err != nil || !ok {
        return 0, err
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return 0, nil
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return 0, err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}

	if err != nil {
		return 0, err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}

    return postgres.RunCountRequestForSchema(ctx, schema, sacQueryFilter, s.db)
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, {{template "paramList" $pks}}) (bool, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "{{.TrimmedType}}")
    {{- end}}{{/* if not $inMigration */}}

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{- if .PermissionChecker }}
    if ok, err := {{ .PermissionChecker }}.ExistsAllowed(ctx); err != nil || !ok {
        return false, err
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return false, nil
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return false, err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
		return false, err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}

    q := search.ConjunctionQuery(
        sacQueryFilter,
    {{- range $idx, $pk := $pks}}
        {{- if eq $pk.Name $singlePK.Name }}
        search.NewQueryBuilder().AddDocIDs({{ $singlePK.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- else }}
        search.NewQueryBuilder().AddExactMatches(search.FieldLabel("{{ searchFieldNameInOtherSchema $pk }}"), {{ $pk.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- end}}
    {{- end}}
    )

	count, err := postgres.RunCountRequestForSchema(ctx, schema, q, s.db)
	// With joins and multiple paths to the scoping resources, it can happen that the Count query for an object identifier
	// returns more than 1, despite the fact that the identifier is unique in the table.
	return count > 0, err
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context, {{template "paramList" $pks}}) (*{{.Type}}, bool, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.GetAllowed(ctx); err != nil || !ok {
        return nil, false, err
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, false, nil
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ" }}
    scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
        return nil, false, err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
        return nil, false, err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}

    q := search.ConjunctionQuery(
    sacQueryFilter,
    {{- range $idx, $pk := $pks}}
        {{- if eq $pk.Name $singlePK.Name }}
            search.NewQueryBuilder().AddDocIDs({{ $singlePK.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- else }}
            search.NewQueryBuilder().AddExactMatches(search.FieldLabel("{{ searchFieldNameInOtherSchema $pk }}"), {{ $pk.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- end}}
    {{- end}}
    )

	data, err := postgres.RunGetQueryForSchema(ctx, schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg {{.Type}}
	if err := proto.Unmarshal(data, &msg); err != nil {
        return nil, false, err
	}
	return &msg, true, nil
}

{{- if .GetAll }}
func(s *storeImpl) GetAll(ctx context.Context) ([]*{{.Type}}, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    var objs []*{{.Type}}
    err := s.Walk(ctx, func(obj *{{.Type}}) error {
        objs = append(objs, obj)
        return nil
    })
    return objs, err
}
{{- end}}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
    {{- if not $inMigration}}
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
    {{- end}}{{/* if not .inMigration */}}
	conn, err := s.db.Acquire(ctx)
	if err != nil {
	    return nil, nil, err
	}
	return conn, conn.Release, nil
}

{{- if not .JoinTable }}
// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context, {{template "paramList" $pks}}) error {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{- if .PermissionChecker }}
    if ok, err := {{ .PermissionChecker }}.DeleteAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.Modify(targetResource))
	if err != nil {
		return err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
		return err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}

    q := search.ConjunctionQuery(
        sacQueryFilter,
    {{- range $idx, $pk := $pks}}
        {{- if eq $pk.Name $singlePK.Name }}
        search.NewQueryBuilder().AddDocIDs({{ $singlePK.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- else }}
        search.NewQueryBuilder().AddExactMatches(search.FieldLabel("{{ searchFieldNameInOtherSchema $pk }}"), {{ $pk.ColumnName|lowerCamelCase }}).ProtoQuery(),
        {{- end}}
    {{- end}}
    )

	return postgres.RunDeleteRequestForSchema(ctx, schema, q, s.db)
}
{{- end}}


{{- if not .JoinTable }}
// DeleteByQuery removes the objects based on the passed query
func (s *storeImpl) DeleteByQuery(ctx context.Context, query *v1.Query) error {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{- if .PermissionChecker }}
    if ok, err := {{ .PermissionChecker }}.DeleteAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.Modify(targetResource))
	if err != nil {
		return err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
		return err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}

    q := search.ConjunctionQuery(
        sacQueryFilter,
        query,
    )

	return postgres.RunDeleteRequestForSchema(ctx, schema, q, s.db)
}
{{- end}}

{{- if $singlePK }}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]{{$singlePK.Type}}, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")
    {{- end}}{{/* if not .inMigration */}}
    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.GetIDsAllowed(ctx); err != nil || !ok {
        return nil, err
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, nil
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return nil, err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
		return nil, err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}
    result, err := postgres.RunSearchRequestForSchema(ctx, schema, sacQueryFilter, s.db)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result))
	for _, entry := range result {
		ids = append(ids, entry.ID)
	}

	return ids, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ctx context.Context, ids []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    if len(ids) == 0 {
        return nil, nil, nil
    }

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{ if .Obj.HasPermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.GetManyAllowed(ctx); err != nil {
        return nil, nil, err
    } else if !ok {
        return nil, nil, nil
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, nil, nil
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
		Resource: targetResource,
		Access:   storage.Access_READ_ACCESS,
	})
	if err != nil {
        return nil, nil, err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
        return nil, nil, err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}
    q := search.ConjunctionQuery(
        sacQueryFilter,
        search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery(),
    )

	rows, err := postgres.RunGetManyQueryForSchema(ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			missingIndices := make([]int, 0, len(ids))
			for i := range ids {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	resultsByID := make(map[{{$singlePK.Type}}]*{{.Type}})
    for _, data := range rows {
		msg := &{{.Type}}{}
		if err := proto.Unmarshal(data, msg); err != nil {
		    return nil, nil, err
		}
		resultsByID[{{$singlePK.Getter "msg"}}] = msg
	}
	missingIndices := make([]int, 0, len(ids)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input ids
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*{{.Type}}, 0, len(resultsByID))
	for i, id := range ids {
		if result, ok := resultsByID[id]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
		    elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}
{{- if .SearchCategory }}
// GetByQuery returns the objects matching the query
func (s *storeImpl) GetByQuery(ctx context.Context, query *v1.Query) ([]*{{.Type}}, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetByQuery, "{{.TrimmedType}}")
    {{- end}}{{/* if not .inMigration */}}

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{ if .Obj.HasPermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.GetManyAllowed(ctx); err != nil {
        return nil, err
    } else if !ok {
        return nil, nil
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, nil
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ" }}
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
		Resource: targetResource,
		Access:   storage.Access_READ_ACCESS,
	})
	if err != nil {
        return nil, err
	}
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
	if err != nil {
        return nil, err
	}
    {{- end }}
    {{- end}}{{/* if not .inMigration */}}
    q := search.ConjunctionQuery(
        sacQueryFilter,
        query,
    )

	rows, err := postgres.RunGetManyQueryForSchema(ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
		    return nil, nil
		}
		return nil, err
	}
	var results []*{{.Type}}
    for _, data := range rows {
		msg := &{{.Type}}{}
		if err := proto.Unmarshal(data, msg); err != nil {
		    return nil, err
		}
		results = append(results, msg)
	}
	return results, nil
}
{{- end }}

{{- if not .JoinTable }}
// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ctx context.Context, ids []{{$singlePK.Type}}) error {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{.TrimmedType}}")
    {{- end }}{{/* if not $inMigration */}}

    var sacQueryFilter *v1.Query
    {{- if not $inMigration}}
    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.DeleteManyAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.Modify(targetResource))
    if err != nil {
        return err
    }
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
    if err != nil {
        return err
    }
    {{- end }}
    {{- end }}{{/* if not $inMigration */}}

    q := search.ConjunctionQuery(
    sacQueryFilter,
        search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery(),
    )

    return postgres.RunDeleteRequestForSchema(ctx, schema, q, s.db)
}
{{- end }}
{{- end }}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *{{.Type}}) error) error {
    var sacQueryFilter *v1.Query
{{- if not $inMigration}}
{{- if .PermissionChecker }}
    if ok, err := {{ .PermissionChecker }}.WalkAllowed(ctx); err != nil || !ok {
        return err
    }
{{- else if .Obj.IsGloballyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil
    }
{{- else if .Obj.IsDirectlyScoped }}
    {{ template "defineScopeChecker" "READ" }}
    scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
        Resource: targetResource,
        Access:   storage.Access_READ_ACCESS,
    })
    if err != nil {
        return err
    }
    {{- if .Obj.IsClusterScope }}
    sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
    {{- else}}
    sacQueryFilter, err = sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    {{- end }}
    if err != nil {
        return err
    }
{{- end }}
{{- end }}{{/* if not $inMigration */}}
	fetcher, closer, err := postgres.RunCursorQueryForSchema(ctx, schema, sacQueryFilter, s.db)
	if err != nil {
		return err
	}
	defer closer()
	for {
		rows, err := fetcher(cursorBatchSize)
		if err != nil {
			return pgutils.ErrNilIfNoRows(err)
		}
		for _, data := range rows {
			var msg {{.Type}}
			if err := proto.Unmarshal(data, &msg); err != nil {
				return err
			}
			if err := fn(&msg); err != nil {
				return err
			}
		}
		if len(rows) != cursorBatchSize {
			break
		}
	}
	return nil
}

//// Used for testing
{{- define "dropTableFunctionName"}}dropTable{{.Table | upperCamelCase}}{{end}}
{{- define "dropTable"}}
{{- $schema := . }}
func {{ template "dropTableFunctionName" $schema }}(ctx context.Context, db *pgxpool.Pool) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{$schema.Table}} CASCADE")
    {{range $child := $schema.Children}}{{ template "dropTableFunctionName" $child }}(ctx, db)
    {{end}}
}
{{range $child := $schema.Children}}{{ template "dropTable" $child }}{{end}}
{{- end}}

{{template "dropTable" .Schema}}

func Destroy(ctx context.Context, db *pgxpool.Pool) {
    {{template "dropTableFunctionName" .Schema}}(ctx, db)
}

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB) Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return New(db)
}

//// Stubs for satisfying legacy interfaces

{{- if eq .TrimmedType "Policy" }}
func (s *storeImpl) RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error {
    return errors.New("unimplemented")
}

func (s *storeImpl) DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error {
    return errors.New("unimplemented")
}
{{- end }}

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}
