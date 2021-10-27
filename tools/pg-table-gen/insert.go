package main

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

type insertionPair struct {
	Elem Element
	SubVar string
}

func generateInsertionPairs(table *Path) []insertionPair {
	var pairs []insertionPair
	for _, elem := range table.Elems {
		pairs = append(pairs, insertionPair{
			Elem: elem,
		})
	}
	for _, child := range table.Children {
		childPairs := generateInsertionPairs(child)
		pairs = append(pairs, childPairs...)
	}
	return pairs
}

func generateTableInsertion(f *File, tableName string, table *Path) {
	objName := strings.ToLower(tableName)

	insertionPairs := generateInsertionPairs(table)

	f.Var().Id("marshaler").Op("=").Op("&").Qual("github.com/gogo/protobuf/jsonpb", "Marshaler{EnumsAsInts: true, EmitDefaults: true}")





	var fmtStmts []Code

	var preprocessingStmts []Code
	for i, p := range insertionPairs {
		if p.Elem.DataType == JSONB {
			subVar := fmt.Sprintf("var%d", i)
			preprocessingStmts = append(preprocessingStmts, generateJSONBField(p, subVar))
			insertionPairs[i].SubVar = subVar
		}
	}

	fmtStmts = append(fmtStmts, Return(Nil()))
	f.Func().Id("Upsert").Params(Id(objName).Op("*").Qual("github.com/stackrox/rox/generated/storage", table.RawFieldType)).Error().Block(
		preprocessingStmts...
	)

	//
	//
	//Upsert(alert *storage.Alert) error
	//
	//
	//
	//var b strings.Builder
	//fmt.Fprintf(&b, "create table if not exists %s (", tableName)
	//fieldsFromPath(&b, "", table)
	//fmt.Fprintf(&b, ")")
	//
	//queryName := fmt.Sprintf("create%sTableQuery", tableName)
	//f.Const().Id(queryName).Op("=").Lit(b.String())
	//
	//f.Func().Id(fmt.Sprintf("Create%sTable", tableName)).Params(Id("db").Op("*").Qual("database/sql", "DB")).Error().Block(
	//	List(Id("_"), Err()).Op(":=").Id("db").Dot("Exec").Call(Id(queryName)),
	//	Return(Err()),
	//)
}
