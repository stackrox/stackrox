package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func tableNeedsSearch(table *Table) bool {
	for _, e := range flattenTable(table) {
		if e.IsSearchable {
			return true
		}
	}
	return false
}

func searchPrint(table *Table) {
	if !tableNeedsSearch(table) {
		return
	}
	fmt.Println(table.TableName())
	for _, elem := range table.Elems {
		if elem.IsSearchable {
			fmt.Println(elem.SQLPath())
		}
	}
	for _, embedded := range table.Embedded {
		searchPrint(embedded)
	}
	fmt.Println()
	for _, children := range table.Children {
		searchPrint(children)
	}
}

func primaryKeyEntry(pks []Element) string {
	var entries []string
	for _, pk := range pks {
		entries = append(entries, pk.SQLPath())
	}

	return fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(entries, ", "))
}

func foreignKeyEntry(table *Table) string {
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

	return fmt.Sprintf("  CONSTRAINT fk_parent_table FOREIGN KEY (%s) REFERENCES %s(%s)", strings.Join(localKeys, ", "), table.Parent.TableName(), strings.Join(parentKeys, ", "))
}

func createTablesForChildren(table *Table) {
	for _, c := range table.Children {
		createTables(c)
	}
}

func createTables(table *Table) {
	if !tableNeedsSearch(table) {
		return
	}

	fmt.Printf("create table if not exists %s(\n", table.TableName())

	if table.Parent != nil {
		for _, elem := range table.PrimaryKeyElements() {
			fmt.Println(" ", elem.SQLPath(), dataTypeToSQLType(elem.DataType), "not null,")
		}
	}
	for _, elem := range flattenTable(table) {
		if !elem.IsSearchable {
			continue
		}
		fmt.Println(" ", elem.SQLPath(), dataTypeToSQLType(elem.DataType), "not null,")
	}
	fmt.Print(primaryKeyEntry(table.PrimaryKeyElements()))
	if fkLine := foreignKeyEntry(table); fkLine != "" {
		fmt.Print(",\n")
		fmt.Println(fkLine)
	} else {
		fmt.Print("\n")
	}
	fmt.Println(");")
	fmt.Println()

	createTablesForChildren(table)
}

func generateTopLevelTable(table *Table) {
	ic := &InsertComposer{}
	ic.AddSQL("serialized")
	ic.AddExcluded("serialized")
	ic.AddGetters("serialized")

	ic.Combine(table.GetInsertComposer())

	fmt.Println(ic.Query())
}

func generateInsertFunctions(table *Table) {
	generateTopLevelTable(table)

	topLevelPks := table.PrimaryKeyElements()

	for _, child := range table.Children {
		generateSubTables(child, topLevelPks, 1)
	}
}

func levelToSpaces(level int) string {
	return strings.Repeat("  ", level)
}

func generateSubTables(table *Table, topLevelPkElems []Element, level int) {
	ic := &InsertComposer{
		Table: table.TableName(),
	}

	for _, elem := range topLevelPkElems {
		ic.AddGetters(elem.GetterPath())
	}
	for i := 0; i < level; i++ {
		ic.AddGetters("idx" + strconv.Itoa(i+1))
	}
	for _, elem := range table.PrimaryKeyElements() {
		ic.AddSQL(elem.SQLPath())
		ic.AddExcluded(elem.SQLPath())
	}

	ic.Combine(table.GetInsertComposer())

	fmt.Println(ic.Query())

	fmt.Printf("for idx%d, obj%d := range %s {\n", level, level, table.GetterPath())
	for _, child := range table.Children {
		generateSubTables(child, topLevelPkElems, level + 1)
	}


}

func insertObject(table *Table, topLevelPkElems []Element, level int) {
	flattenedElements := flattenTable(table)

	isRootTable := level == 0

	var sqlValues []string
	var excludedValues []string
	var getterPaths []string
	var placeholders []string
	var placeHolderCount int

	if isRootTable {
		sqlValues = append(sqlValues, "serialized")
		excludedValues = append(excludedValues, "EXCLUDED.serialized = serialized")
		getterPaths = append(getterPaths, "value")
		placeholders = append(placeholders, "$1")
		placeHolderCount = 1
	} else {
		for i, elem := range table.PrimaryKeyElements() {
			path := elem.SQLPath()
			sqlValues = append(sqlValues, path)
			excludedValues = append(excludedValues, fmt.Sprintf("EXCLUDED.%s = %s", path, path))
			placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
			placeHolderCount++
		}
		fmt.Println(sqlValues)
		fmt.Println()
	}

	for i, elem := range flattenedElements {
		sqlValues = append(sqlValues, elem.SQLPath())
		excludedValues = append(excludedValues, "EXCLUDED." + elem.SQLPath() + " = " + elem.SQLPath())
		getterPaths = append(getterPaths, elem.GetterPath())
		placeholders = append(placeholders, "$"+strconv.Itoa(placeHolderCount+i+1))
	}

	sqlValueStr := strings.Join(sqlValues, ", ")
	excludedValueStr := strings.Join(excludedValues, ", ")
	//getterPathStr := strings.Join(getterPaths, ", ")
	placeholderStr := strings.Join(placeholders, ", ")
	// insert

	query := fmt.Sprintf("insert into %s(%s) values(%s) on conflict update %s", table.TableName(), sqlValueStr, placeholderStr, excludedValueStr)
		fmt.Printf("for idx%d, subObj%d := range %s {\n", level, level, table.GetterPath())
		fmt.Println("  ", query)
		for _, child := range table.Children {
			insertObject(child, nil, level+1)
		}
		fmt.Println()
		fmt.Println("}")
}

func TestWalker(t *testing.T) {
	//Walk(reflect.TypeOf((*storage.Deployment)(nil)))
	table := Walk(reflect.TypeOf((*storage.Deployment)(nil)))

	//table.Print("", true)
	//searchPrint(table)
	//createTables(table)
	//insertObject(table, nil, 0)
	generateInsertFunctions(table)
}

/*
	_, err = tx.Exec(context.Background(), "delete from container_normalized where parent_deployment_id = $1 and container_idx >= $2", dep.GetId(), len(dep.GetContainers()))
 */

