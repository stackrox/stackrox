package main

import (
	"fmt"
	"strings"
)

func fieldDeclaration(name string, dataType DataType) string {
	var sqlType string
	switch dataType {
	case BOOL:
		sqlType = "bool"
	case NUMERIC:
		sqlType = "numeric"
	case STRING:
		sqlType = "varchar"
	case DATETIME:
		sqlType = "timestamp"
	case MAP:
		sqlType = "jsonb"
	case ENUM:
		sqlType = "integer"
	case STRING_ARRAY:
		sqlType = "text[][]"
	default:
		panic(dataType.String())
	}
	return name + " " + sqlType
}

func generateTableDeclarations(parentTable, table *Table) {
	header := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table.Name)

	var pks []string
	var fields []string
	var fkNames []string
	var fkRelationships []string

	for _, field := range table.ForeignKeys {
		localName := "parent_" + field.name
		localName = normalizeName(localName)
		fkNames = append(fkNames, localName)
		fkRelationships = append(fkRelationships, field.name)

		pks = append(pks, localName)

		line := fieldDeclaration(localName, field.datatype)
		fields = append(fields, line)
	}

	for _, field := range table.Fields {
		line := fieldDeclaration(field.name, field.datatype)
		fields = append(fields, line)
		if field.pk {
			pks = append(pks, field.name)
		}
	}
	// add primary key field
	if len(fkNames) > 0 {
		fields = append(fields, fmt.Sprintf("foreign key (%s) references %s(%s)", strings.Join(fkNames, ", "), parentTable.Name, strings.Join(fkRelationships, ", ")))
	}
	fields = append(fields, fmt.Sprintf("primary key (%s)", strings.Join(pks, ", ")))


	for i := range fields {
		fields[i] = "\t" + fields[i]
	}

	body := strings.Join(fields, ",\n")
	footer := ");"

	fmt.Print(header)
	fmt.Print(body)
	fmt.Print(footer)
	fmt.Println()

	for _, child := range table.ChildTables {
		generateTableDeclarations(table, child)
	}
}
