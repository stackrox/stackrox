package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/walker"
)

func generateTopLevelTable(w io.Writer, table *walker.Table) {
	ic := &walker.InsertComposer{
		Table: table.TableName(),
	}
	ic.AddSQL("serialized")
	ic.AddExcluded("serialized")
	ic.AddGetters("serialized")

	for _, elem := range table.PrimaryKeyElements() {
		ic.AddPK(elem.SQLPath())
	}
	ic.Combine(table.GetInsertComposer(0))

	fmt.Fprintf(w, "localQuery := \"%s\"\n", ic.Query())
	fmt.Fprintf(w,"_, err = tx.Exec(context.Background(), localQuery, %s)\n", ic.ExecGetters())
	fmt.Fprint(w,"if err != nil {\n    return err\n  }\n")
}

func generateInsertFunctions(table *walker.Table) string {
	buf := bytes.NewBuffer(nil)

	generateTopLevelTable(buf, table)

	topLevelPks := table.PrimaryKeyElements()

	for _, child := range table.Children {
		generateSubTables(buf, child, topLevelPks, 1)
	}
	return buf.String()
}

func levelToSpaces(level int) string {
	return strings.Repeat("  ", level)
}

func generateSubTables(w io.Writer, table *walker.Table, topLevelPkElems []walker.Element, level int) {
	if !table.TopLevel {
		for _, c := range table.Children {
			generateSubTables(w, c, topLevelPkElems, level)
		}
		for _, e := range table.Embedded {
			generateSubTables(w, e, topLevelPkElems, level)
		}
		return
	}
	if !tableNeedsSearch(table){
		return
	}
	ic := &walker.InsertComposer{
		Table: table.TableName(),
	}

	var pkGetters []string
	for _, elem := range topLevelPkElems {
		getter := "obj0." + elem.GetterPath()
		ic.AddGetters(getter)
		pkGetters = append(pkGetters, getter)
	}
	for i := 0; i < level; i++ {
		idx := "idx" + strconv.Itoa(i+1)
		ic.AddGetters(idx)
		if i != level - 1 {
			pkGetters = append(pkGetters, idx)
		}
	}
	var deleteFromClauses []string
	pkElements := table.PrimaryKeyElements()
	for i, elem := range pkElements {
		sqlPath := elem.SQLPath()
		ic.AddSQL(sqlPath)
		ic.AddPK(sqlPath)
		ic.AddExcluded(sqlPath)
		if i != len(pkElements) - 1 {
			deleteFromClauses = append(deleteFromClauses, fmt.Sprintf("%s = $%d", sqlPath, i+1))
		}
	}

	ic.Combine(table.GetInsertComposer(level))

	sliceGetter := fmt.Sprintf("obj%d.%s", level-1, table.AbsGetterPath())

	fmt.Fprintf(w, "%sfor idx%d, obj%d := range %s {\n", levelToSpaces(level), level, level, sliceGetter)
	fmt.Fprintf(w, "  %slocalQuery := \"%s\"\n", levelToSpaces(level), ic.Query())
	fmt.Fprintf(w,"  %s_, err := tx.Exec(context.Background(), localQuery, %s)\n", levelToSpaces(level), ic.ExecGetters())
	fmt.Fprintf(w,"  %sif err != nil {\n    %sreturn err\n  %s}\n", levelToSpaces(level), levelToSpaces(level), levelToSpaces(level))
	for _, child := range table.Children {
		generateSubTables(w, child, topLevelPkElems, level + 1)
	}
	for _, e := range table.Embedded {
		generateSubTables(w, e, topLevelPkElems, level + 1)
	}
	fmt.Fprintf(w,"%s}\n", levelToSpaces(level))
	fmt.Fprintf(w,"  %s_, err = tx.Exec(context.Background(), \"delete from %s where %s and idx >= $%d\", %s)\n", levelToSpaces(level), table.TableName(), strings.Join(deleteFromClauses, " and "), len(deleteFromClauses)+1, strings.Join(append(pkGetters, fmt.Sprintf("len(%s)", sliceGetter)), ", "))
	fmt.Fprintf(w,"  %sif err != nil {\n    %sreturn err\n  %s}\n", levelToSpaces(level), levelToSpaces(level), levelToSpaces(level))
}
