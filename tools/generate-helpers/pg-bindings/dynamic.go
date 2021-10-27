package main

import (
	"fmt"
	"strings"
)

func flattenTable(table *Path) []Element {
	var elems []Element
	for _, elem := range table.Elems {
		if !elem.IsSearchable {
			continue
		}
		if strings.EqualFold(elem.SQLPath(), "id") {
			continue
		}
		if elem.DataType == STRING || elem.DataType == NUMERIC {
			elems = append(elems, elem)
		}
	}
	for _, child := range table.Children {
		childPairs := flattenTable(child)
		elems = append(elems, childPairs...)
	}
	return elems
}

func generateTableCreationQuery(tableName string, elements []Element) string {
	fields := []string {
		"id varchar primary key",
		"value jsonb",
	}
	for _, elem := range elements {
		fields = append(fields, fmt.Sprintf("%s %s", elem.SQLPath(), dataTypeToSQLType(elem.DataType)))
	}
	return fmt.Sprintf("create table if not exists %s (%s)", tableName, strings.Join(fields, ", "))
}

func generateTableInsertionQuery(tableName string, elements []Element) (string, string) {
	fields := []string {
		"id",
		"value",
	}
	for _, elem := range elements {
		fields = append(fields, elem.SQLPath())
	}

	var excludedFields []string
	for _, field := range fields[1:] {
		excludedFields = append(excludedFields, fmt.Sprintf("%s = EXCLUDED.%s", field, field))
	}

	var valuePlaceholders []string
	for i := range fields {
		valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("$%d", i+1))
	}

	valueGetters := []string {
		"id",
		"value",
	}
	for _, elem := range elements {
		valueGetters = append(valueGetters, "obj." + elem.GetterPath())
	}

	return fmt.Sprintf("insert into %s (%s) values(%s) on conflict(id) do update set %s",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(valuePlaceholders, ", "),
		strings.Join(excludedFields, ", "),
	), strings.Join(valueGetters, ", ")
}
