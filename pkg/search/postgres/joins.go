package postgres

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

type JoinType int

const (
	Inner JoinType = iota
	Left
)

var (
	schemasWithImageID    = []*walker.Schema{schema.ImagesSchema, schema.ImageComponentEdgesSchema}
	schemasWithImageCVEID = []*walker.Schema{schema.ImageCvesSchema, schema.ImageComponentCveEdgesSchema}
)

type joinTreeNode struct {
	currNode *walker.Schema
	// Specifies how this node will join with the parent node
	joinType JoinType
	children map[*joinTreeNode][]walker.ColumnNamePair
}

// addPathToTree adds the path to the tree.
// Expected invariants:
// * The first element of path MUST correspond to the same table as this node.
// * Path must NOT be empty.
// * type of Join along new nodes that will be created when adding this path to the tree. Should be either Inner or Left.
func (j *joinTreeNode) addPathToTree(path []joinPathElem, finalNode *walker.Schema, joinType JoinType) {
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
			if joinType == Inner {
				child.joinType = Inner
			}
			relevantChild = child
			break
		}
	}
	if relevantChild == nil {
		relevantChild = &joinTreeNode{
			currNode: nextNode,
			joinType: joinType,
		}
		if j.children == nil {
			j.children = make(map[*joinTreeNode][]walker.ColumnNamePair)
		}
		j.children[relevantChild] = first.columnPairs
	}
	if len(rest) > 0 {
		relevantChild.addPathToTree(rest, finalNode, joinType)
	}
}

// toJoins walks the tree to construct the linearized set of inner joins that we need to do.
func (j *joinTreeNode) toJoins() []Join {
	joins := make([]Join, 0)

	if j == nil {
		return joins
	}

	j.appendJoinsHelper(&joins)
	return joins
}

func (j *joinTreeNode) appendJoinsHelper(joins *[]Join) {
	// Ensure the joins are added in a deterministic order to the query
	// for testing purposes.
	children := make([]*joinTreeNode, 0, len(j.children))
	for child := range j.children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool { return children[i].currNode.Table < children[j].currNode.Table })
	for _, child := range children {
		columnPairs := j.children[child]
		*joins = append(*joins, Join{
			leftTable:       j.currNode.Table,
			rightTable:      child.currNode.Table,
			joinType:        child.joinType,
			columnNamePairs: columnPairs,
		})
		child.appendJoinsHelper(joins)
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

func collectFields(q *v1.Query) (set.StringSet, set.StringSet) {
	var queries []*v1.Query
	collectedFields := set.NewStringSet()
	nullableFields := set.NewStringSet()
	switch sub := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		switch subBQ := q.GetBaseQuery().Query.(type) {
		case *v1.BaseQuery_DocIdQuery, *v1.BaseQuery_MatchNoneQuery:
			// nothing to do
		case *v1.BaseQuery_MatchFieldQuery:
			collectedFields.Add(subBQ.MatchFieldQuery.GetField())
			if subBQ.MatchFieldQuery.GetValue() == search.NullString {
				nullableFields.Add(subBQ.MatchFieldQuery.GetField())
			}
		case *v1.BaseQuery_MatchLinkedFieldsQuery:
			for _, q := range subBQ.MatchLinkedFieldsQuery.Query {
				collectedFields.Add(q.GetField())
				if q.GetValue() == search.NullString {
					nullableFields.Add(q.GetField())
				}
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
		collected, nullable := collectFields(query)
		collectedFields.AddAll(collected.AsSlice()...)
		nullableFields.AddAll(nullable.AsSlice()...)
	}
	for _, selectField := range q.GetSelects() {
		collectedFields.Add(selectField.GetField().GetName())
		collected, nullable := collectFields(selectField.GetFilter().GetQuery())
		collectedFields.AddAll(collected.AsSlice()...)
		nullableFields.AddAll(nullable.AsSlice()...)
	}
	for _, groupByField := range q.GetGroupBy().GetFields() {
		collectedFields.Add(groupByField)
	}
	for _, sortOption := range q.GetPagination().GetSortOptions() {
		collectedFields.Add(sortOption.GetField())
	}
	return collectedFields, nullableFields
}

type searchFieldMetadata struct {
	baseField       *walker.Field
	derivedMetadata *walker.DerivedSearchField
}

func getJoinsAndFields(src *walker.Schema, q *v1.Query) ([]Join, map[string]searchFieldMetadata) {
	unreachedFields, nullableFields := collectFields(q)

	if env.ImageCVEEdgeCustomJoin.BooleanSetting() && !features.FlattenCVEData.Enabled() {
		// Step 1: If ImageCveEdgesSchema is going to be a part of joins, we want to ensure that we are able to join on both
		//  ImageId and ImageCveId fields
		if src != schema.ImageCveEdgesSchema &&
			containsFieldsFromSchemas(unreachedFields, []*walker.Schema{schema.ImageCveEdgesSchema}) {
			if !containsFieldsFromSchemas(unreachedFields, schemasWithImageID) {
				unreachedFields.Add(strings.ToLower(search.ImageSHA.String()))
			}
			if !containsFieldsFromSchemas(unreachedFields, schemasWithImageCVEID) {
				unreachedFields.Add(strings.ToLower(search.CVEID.String()))
			}
		}
	}

	joinTreeRoot := &joinTreeNode{
		currNode: src,
	}
	reachableFields := make(map[string]searchFieldMetadata)
	queue := []bfsQueueElem{{schema: src}}
	visited := set.NewStringSet()

	imageCveEdgesSchemaInQueue := false
	if src == schema.ImageCveEdgesSchema {
		imageCveEdgesSchemaInQueue = true
	}

	for len(queue) > 0 && len(unreachedFields) > 0 {
		currElem := queue[0]
		queue = queue[1:]

		if env.ImageCVEEdgeCustomJoin.BooleanSetting() && !features.FlattenCVEData.Enabled() {
			// Step 2: Avoid using ImageCveEdgesSchema unless there is no other way to get to the required fields.
			// If ImageCveEdgesSchema is root schema, then it is unavoidable.
			if currElem.schema == schema.ImageCveEdgesSchema && currElem.schema != src {
				// If there are more schemas to expand in queue, expand them first. Hopefully we will find all required fields
				//   without having to use ImageCveEdgesSchema.
				// But if there are no more schemas to expand, and we still have unreachable fields,
				//   then they must be fields of ImageCveEdgesSchema. In that case ImageCveEdgesSchema is unavoidable.
				if len(queue) > 0 {
					queue = append(queue, currElem)
					continue
				}
			}
		}

		if !visited.Add(currElem.schema.Table) {
			continue
		}
		numReachableFieldsBefore := len(reachableFields)
		joinType := Inner
		for _, f := range currElem.schema.Fields {
			field := f
			if !f.Derived {
				lowerCaseName := strings.ToLower(f.Search.FieldName)
				if unreachedFields.Remove(lowerCaseName) {
					reachableFields[lowerCaseName] = searchFieldMetadata{baseField: &field}
					if nullableFields.Remove(lowerCaseName) {
						joinType = Left
					}
				}
			}

			for _, derivedF := range field.DerivedSearchFields {
				derivedField := derivedF
				lowerCaseDerivedName := strings.ToLower(derivedField.DerivedFrom)
				if unreachedFields.Remove(lowerCaseDerivedName) {
					reachableFields[lowerCaseDerivedName] = searchFieldMetadata{
						baseField:       &field,
						derivedMetadata: &derivedField,
					}
				}
				if nullableFields.Remove(lowerCaseDerivedName) {
					joinType = Left
				}
			}
		}
		// We found a field in this schema; if this is not the root schema itself, we'll need to add it to the join tree.
		if len(reachableFields) > numReachableFieldsBefore && len(currElem.pathFromRoot) > 0 {
			joinTreeRoot.addPathToTree(currElem.pathFromRoot, currElem.schema, joinType)
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

			if env.ImageCVEEdgeCustomJoin.BooleanSetting() && !features.FlattenCVEData.Enabled() {
				// We want to make sure ImageCveEdgesSchema gets added only once to queue. If there are multiple copies of
				// ImageCveEdgesSchema in the queue, then we can enter an infinite loop trying to push one copy after another
				// to the end of queue.
				if rel.OtherSchema == schema.ImageCveEdgesSchema {
					if imageCveEdgesSchemaInQueue {
						continue
					} else {
						imageCveEdgesSchemaInQueue = true
					}
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
			if src.SearchScope == nil {
				queue = append(queue, newElem)
			} else if _, foundInSearchScope := src.SearchScope[newElem.schema.OptionsMap.PrimaryCategory()]; foundInSearchScope {
				queue = append(queue, newElem)
			}
		}
	}

	joinTreeRoot.removeUnnecessaryRelations(reachableFields)

	return joinTreeRoot.toJoins(), reachableFields
}

func containsFieldsFromSchemas(fields set.StringSet, schemas []*walker.Schema) bool {
	for _, s := range schemas {
		for _, f := range s.Fields {
			if !f.Derived {
				if fields.Contains(strings.ToLower(f.Search.FieldName)) {
					return true
				}
			}
			for _, derivedF := range f.DerivedSearchFields {
				if fields.Contains(strings.ToLower(derivedF.DerivedFrom)) {
					return true
				}
			}
		}
	}
	return false
}

// removeUnnecessaryRelations removes inner join tables where the same column
// is used by the previous and next table in the join chain. i.e.
// a INNER JOIN b ON a.id = b.same_column
// b INNER JOIN c ON b.same_column = c.id
// If table b is not used in any other way, we can remove it from the join chain.
// Outcome: a INNER JOIN c ON a.id = c.id
func (j *joinTreeNode) removeUnnecessaryRelations(requiredFields map[string]searchFieldMetadata) {
	if j == nil {
		return
	}

	requiredTables := set.NewSet[string]()
	for _, fieldMetadata := range requiredFields {
		requiredTables.Add(fieldMetadata.baseField.Schema.Table)
	}

	rootChildren := make(map[*joinTreeNode][]walker.ColumnNamePair)
	for child, columnPairs := range j.children {
		child.removeUnnecessaryRelations(requiredFields)

		if requiredTables.Contains(child.currNode.Table) {
			rootChildren[child] = columnPairs

			continue
		}

		childColumns := make(map[string]string, len(columnPairs))
		for _, pair := range columnPairs {
			childColumns[pair.ColumnNameInOtherSchema] = pair.ColumnNameInThisSchema
		}

		childChildren := make(map[*joinTreeNode][]walker.ColumnNamePair)
		for childChild, childColumnPairs := range child.children {
			if len(columnPairs) != len(childColumnPairs) {
				childChildren[childChild] = childColumnPairs

				continue
			}

			rootColumnPairs := make([]walker.ColumnNamePair, 0, len(childColumnPairs))
			for _, childColumnPair := range childColumnPairs {
				if _, found := childColumns[childColumnPair.ColumnNameInThisSchema]; !found {
					break
				}

				rootColumnPairs = append(rootColumnPairs, walker.ColumnNamePair{
					ColumnNameInThisSchema:  childColumns[childColumnPair.ColumnNameInThisSchema],
					ColumnNameInOtherSchema: childColumnPair.ColumnNameInOtherSchema,
				})
			}

			if len(columnPairs) == len(rootColumnPairs) {
				rootChildren[childChild] = rootColumnPairs
			} else {
				childChildren[childChild] = childColumnPairs
			}
		}

		// Remove the table because all next tables are paired with the previous table.
		if len(child.children) != 0 && len(childChildren) == 0 {
			continue
		}

		child.children = childChildren
		rootChildren[child] = columnPairs
	}

	j.children = rootChildren
}
