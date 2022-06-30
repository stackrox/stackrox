package postgres

import (
	"fmt"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/set"
)

type joinTreeNode struct {
	currNode *walker.Schema
	children map[*joinTreeNode][]walker.ColumnNamePair
}

// addPathToTree adds the path to the tree.
// Expected invariants:
// * The first element of path MUST correspond to the same table as this node.
// * Path must NOT be empty.
func (j *joinTreeNode) addPathToTree(path []joinPathElem, finalNode *walker.Schema) {
	first, rest := path[0], path[1:]
	if first.table != j.currNode {
		panic(fmt.Sprintf("unexpected error in join tree node construction: node is %+v, path is %+v", j, path))
	}
	var nextNode *walker.Schema
	if len(rest) == 0 {
		nextNode = finalNode
	} else {
		nextNode = rest[0].table
	}
	var relevantChild *joinTreeNode
	for child := range j.children {
		if child.currNode == nextNode {
			relevantChild = child
			break
		}
	}
	if relevantChild == nil {
		relevantChild = &joinTreeNode{
			currNode: nextNode,
		}
		if j.children == nil {
			j.children = make(map[*joinTreeNode][]walker.ColumnNamePair)
		}
		j.children[relevantChild] = first.columnPairs
	}
	if len(rest) > 0 {
		relevantChild.addPathToTree(rest, finalNode)
	}
}

// toInnerJoins walks the tree to construct the linearized set of inner joins that we need to do.
func (j *joinTreeNode) toInnerJoins() []innerJoin {
	innerJoins := make([]innerJoin, 0)
	j.appendInnerJoinsHelper(&innerJoins)
	return innerJoins
}

func (j *joinTreeNode) appendInnerJoinsHelper(joins *[]innerJoin) {
	for child, columnPairs := range j.children {
		*joins = append(*joins, innerJoin{
			leftTable:       j.currNode.Table,
			rightTable:      child.currNode.Table,
			columnNamePairs: columnPairs,
		})
		child.appendInnerJoinsHelper(joins)
	}
}

type joinPathElem struct {
	table       *walker.Schema
	columnPairs []walker.ColumnNamePair
}

type bfsQueueElem struct {
	schema       *walker.Schema
	pathFromRoot []joinPathElem
}

func collectFields(q *v1.Query) set.StringSet {
	var queries []*v1.Query
	collectedFields := set.NewStringSet()
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery, *v1.BaseQuery_MatchNoneQuery:
			// nothing to do
		case *v1.BaseQuery_MatchFieldQuery:
			collectedFields.Add(subBQ.MatchFieldQuery.GetField())
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				collectedFields.Add(q.GetField())
			}
		default:
			panic("unsupported")
		}
	case *v1.Query_Conjunction:
		queries = append(queries, sub.Conjunction.Queries...)
	case *v1.Query_Disjunction:
		queries = append(queries, sub.Disjunction.Queries...)
	case *v1.Query_BooleanQuery:
		queries = append(queries, sub.BooleanQuery.Must.Queries...)
		queries = append(queries, sub.BooleanQuery.MustNot.Queries...)
	}

	for _, query := range queries {
		collectedFields.AddAll(collectFields(query).AsSlice()...)
	}
	for _, sortOption := range q.GetPagination().GetSortOptions() {
		collectedFields.Add(sortOption.GetField())
	}
	return collectedFields
}

type searchFieldMetadata struct {
	baseField       *walker.Field
	derivedMetadata *walker.DerivedSearchField
}

func getJoinsAndFields(src *walker.Schema, q *v1.Query) ([]innerJoin, map[string]searchFieldMetadata) {
	unreachedFields := collectFields(q)
	joinTreeRoot := &joinTreeNode{
		currNode: src,
	}
	reachableFields := make(map[string]searchFieldMetadata)
	queue := []bfsQueueElem{{schema: src}}
	visited := set.NewStringSet()
	for len(queue) > 0 && len(unreachedFields) > 0 {
		currElem := queue[0]
		queue = queue[1:]
		if !visited.Add(currElem.schema.Table) {
			continue
		}
		numReachableFieldsBefore := len(reachableFields)
		for _, f := range currElem.schema.Fields {
			field := f
			lowerCaseName := strings.ToLower(f.Search.FieldName)
			if unreachedFields.Remove(lowerCaseName) {
				reachableFields[lowerCaseName] = searchFieldMetadata{baseField: &field}
			}
			for _, derivedF := range field.DerivedSearchFields {
				derivedField := derivedF
				lowerCaseDerivedName := strings.ToLower(derivedField.FieldName)
				if unreachedFields.Remove(lowerCaseDerivedName) {
					reachableFields[lowerCaseDerivedName] = searchFieldMetadata{
						baseField:       &field,
						derivedMetadata: &derivedField,
					}
				}
			}
		}
		// We found a field in this schema; if this is not the root schema itself, we'll need to add it to the join tree.
		if len(reachableFields) > numReachableFieldsBefore && len(currElem.pathFromRoot) > 0 {
			joinTreeRoot.addPathToTree(currElem.pathFromRoot, currElem.schema)
		}

	allRelationshipsLoop:
		for _, rel := range currElem.schema.AllRelationships() {
			// Don't go back to something we've already seen in this path.
			// This is not strictly required since the visited check above will take care of this case too,
			// but it is cleaner and will save some work.
			for _, elemInPath := range currElem.pathFromRoot {
				if elemInPath.table == rel.OtherSchema {
					continue allRelationshipsLoop
				}
			}
			newElem := bfsQueueElem{
				schema: rel.OtherSchema,
			}
			newElem.pathFromRoot = make([]joinPathElem, len(currElem.pathFromRoot)+1)
			copy(newElem.pathFromRoot, currElem.pathFromRoot)
			newElem.pathFromRoot[len(newElem.pathFromRoot)-1] = joinPathElem{
				table:       currElem.schema,
				columnPairs: rel.MappedColumnNames,
			}
			if searchScope := searchScope[src.OptionsMap.PrimaryCategory()]; searchScope == nil {
				queue = append(queue, newElem)
			} else if _, foundInSearchScope := searchScope[newElem.schema.OptionsMap.PrimaryCategory()]; foundInSearchScope {
				queue = append(queue, newElem)
			}
		}
	}

	return joinTreeRoot.toInnerJoins(), reachableFields
}
