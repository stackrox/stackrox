package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/walker"
)

func tableNeedsSearch(table *walker.Table) bool {
	for _, e := range table.Elements() {
		if e.IsSearchable() {
			return true
		}
	}
	return false
}

func primaryKeyEntry(pks []walker.Element) string {
	var entries []string
	for _, pk := range pks {
		entries = append(entries, pk.SQLPath())
	}

	return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(entries, ", "))
}

func foreignKeyEntry(table *walker.Table) string {
	if table.Parent == nil {
		return ""
	}
	var localKeys []string
	localPrimaryKeys := table.PrimaryKeyElements()
	// chops off the idx field
	for _, elem := range localPrimaryKeys[:len(localPrimaryKeys)-1] {
		localKeys = append(localKeys, elem.SQLPath())
	}

	var parentKeys []string
	for _, elem := range table.Parent.PrimaryKeyElements() {
		parentKeys = append(parentKeys, elem.SQLPath())
	}

	return fmt.Sprintf("CONSTRAINT fk_parent_table FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE CASCADE", strings.Join(localKeys, ", "), table.Parent.TableName(), strings.Join(parentKeys, ", "))
}

func createTablesForChildren(table *walker.Table) []string {
	var creates []string
	for _, c := range table.Children {
		creates = append(creates, createTables(c)...)
	}
	for _, e := range table.Embedded {
		creates = append(creates, createTablesForChildren(e)...)
	}
	return creates
}

func generateIndexes(table *walker.Table) []string {
	var indexes []string
	for _, elem := range table.Elements() {
		if idx := elem.Options.Index; idx != "" {
			index := fmt.Sprintf("create index if not exists %s_%s on %s using %s(%s)", table.TableName(), elem.SQLPath(), table.TableName(), idx, elem.SQLPath())
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func createTables(table *walker.Table) []string {
	if table.Parent != nil && !(tableNeedsSearch(table) || table.TopLevel) {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "create table if not exists %s(", table.TableName())

	if table.Parent != nil {
		for _, elem := range table.PrimaryKeyElements() {
			fmt.Fprintf(buf,"%s %s not null, ", elem.SQLPath(), walker.DataTypeToSQLType(elem.DataType))
		}
	} else {
		fmt.Fprint(buf,"serialized jsonb not null, ")
	}
	for _, elem := range table.Elements() {
		if !elem.IsSearchable() {
			continue
		}
		fmt.Fprintf(buf,"%s %s, ", elem.SQLPath(), walker.DataTypeToSQLType(elem.DataType))
	}
	fmt.Fprint(buf, primaryKeyEntry(table.PrimaryKeyElements()))
	if fkLine := foreignKeyEntry(table); fkLine != "" {
		fmt.Fprint(buf, ", ", fkLine)
	}
	fmt.Fprint(buf, ");")

	tables := []string {
		buf.String(),
	}
	tables = append(tables, generateIndexes(table)...)

	childTables := createTablesForChildren(table)
	return append(tables, childTables...)
}
