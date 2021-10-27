package old
//
//import (
//	"bytes"
//	"database/sql"
//	"fmt"
//	"os"
//	"strings"
//	"text/template"
//
//	"github.com/stackrox/rox/generated/storage"
//)
//
//func FieldsToQueries(fields []Field) string {
//	fieldNames := make([]string, 0, len(fields))
//	variablePlaceholders := make([]string, 0, len(fieldNames))
//	for i, f := range fields {
//		fieldNames = append(fieldNames, f.NormalizedName())
//		variablePlaceholders = append(variablePlaceholders, fmt.Sprintf("$%d", i+1))
//	}
//	return fmt.Sprintf("(%s) VALUES(%s)", strings.Join(fieldNames, ", "), strings.Join(variablePlaceholders, ", "))
//}
//
//func ToGetter(s string) string {
//	spl := strings.Split(s, ".")
//	for i := range spl {
//		spl[i] = fmt.Sprintf("Get%s()", spl[i])
//	}
//	return strings.Join(spl, ".")
//}
//
//func PathToFn(path string) string {
//	return strings.ReplaceAll(path, ".", "_")
//}
//
//func genInsertion(table *Table) {
//	funcMap := template.FuncMap{
//		"ToLower":         strings.ToLower,
//		"ToGetter":        ToGetter,
//		"FieldsToQueries": FieldsToQueries,
//		"PathToFn":        PathToFn,
//	}
//
//	tmplStr, _ := os.ReadFile("/Users/connorgorman/repos/src/github.com/stackrox/rox/tools/pg-table-gen/templates/insert.tmpl")
//
//	tmpl, err := template.New("insertionTemplate").Funcs(funcMap).Parse(string(tmplStr))
//	if err != nil {
//		panic(err)
//	}
//	var result bytes.Buffer
//
//	if err := tmpl.Execute(&result, table); err != nil {
//		panic(err)
//	}
//	fmt.Println(result.String())
//
//	for _, t := range table.ChildTables {
//		genInsertion(t)
//	}
//
//	//for _, t := range table {
//	//}
//}
//
//func insert(tx *sql.Tx, deployment *storage.Deployment) error {
//	// Insert deployment
//	// iterate over containers
//	// add container fields
//
//	return nil
//}
