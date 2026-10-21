package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

func parseReferencesAndInjectPeerSchemas(schema *walker.Schema, refs []string) (parsedRefs []parsedReference) {
	schemasByObjType := make(map[string]*walker.Schema, len(refs))
	parsedRefs = make([]parsedReference, 0, len(refs))
	for _, ref := range refs {
		var refTable, refObjType string
		if strings.Contains(ref, ":") {
			refTable, refObjType = stringutils.Split2(ref, ":")
		} else {
			refObjType = ref
			refTable = pgutils.NamingStrategy.TableName(stringutils.GetAfter(refObjType, "."))
		}

		refMsgType := protoutils.MessageType(refObjType)
		if refMsgType == nil {
			log.Fatalf("could not find message for type: %s", refObjType)
		}
		refSchema := walker.Walk(refMsgType, refTable)
		schemasByObjType[refObjType] = refSchema
		parsedRefs = append(parsedRefs, parsedReference{
			TypeName: refObjType,
			Table:    refTable,
		})
	}
	schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
		return schemasByObjType[fmt.Sprintf("storage.%s", messageTypeName)]
	})
	return parsedRefs
}

func splitWords(s string) []string {
	var words []string
	var currWord strings.Builder

	for _, r := range s {
		newWord, skip := false, false
		if unicode.IsUpper(r) && currWord.Len() > 0 {
			newWord = true
		} else if !unicode.IsOneOf([]*unicode.RangeTable{unicode.Letter, unicode.Digit}, r) {
			newWord, skip = true, true
		}

		if newWord && currWord.Len() > 0 {
			words = append(words, currWord.String())
			currWord.Reset()
		}
		if !skip {
			currWord.WriteRune(r)
		}
	}

	if currWord.Len() > 0 {
		words = append(words, currWord.String())
	}

	return words
}

func applyPointwise(ss []string, f func(string) string) {
	for i, s := range ss {
		ss[i] = f(s)
	}
}

func lowerCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	words[0] = strings.ToLower(words[0])
	if len(words) == 1 && words[0] == "id" {
		return "id"
	}
	applyPointwise(words[1:], strings.Title)
	applyPointwise(words, stringutils.UpperCaseAcronyms)
	return strings.Join(words, "")
}

func upperCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	applyPointwise(words, strings.Title)
	applyPointwise(words, stringutils.UpperCaseAcronyms)
	return strings.Join(words, "")
}

func valueExpansion(size int) string {
	var all []string
	for i := range size {
		all = append(all, fmt.Sprintf("$%d", i+1))
	}
	return strings.Join(all, ", ")
}

func concatWith(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func arr(els ...any) []any {
	return els
}

func isMessageBytes(f walker.Field) bool {
	return f.DataType == postgres.MessageBytes
}

type subMsgInit struct {
	SetterPath string
	GoType     string
}

func subMessageInits(schema *walker.Schema) []subMsgInit {
	var inits []subMsgInit
	for path, typ := range schema.SubMessages {
		inits = append(inits, subMsgInit{SetterPath: "obj." + path, GoType: typ})
	}
	return inits
}

func messageBytesElemType(f walker.Field) string {
	t := f.Type
	if strings.HasPrefix(t, "[]") {
		return t[2:]
	}
	return t
}

// IndexInfo holds the data needed to render a postgres.IndexDefinition literal in the template.
type IndexInfo struct {
	Name      string
	CreateSQL string
}

var safeIdentifier = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// indexColumn is a single column participating in an index. priority comes from the
// optional index `priority:` option; order is the column's position in proto field
// order. Columns are sorted by (priority asc, order asc) at build time, so columns
// without an explicit priority (sentinel noIndexPriority) sort after prioritized ones
// while keeping field order among themselves.
type indexColumn struct {
	name     string
	priority int
	order    int
}

// noIndexPriority is the sentinel used for columns that declare no `priority:` option.
// It sorts after any real priority, so indexes that never use priority stay byte-identical.
const noIndexPriority = math.MaxInt32

// indexBuilder accumulates columns for a single index (which may span multiple fields
// in the case of composite indexes) and produces the final IndexInfo.
type indexBuilder struct {
	name      string
	table     string
	columns   []indexColumn
	indexType string
}

func (b *indexBuilder) addColumn(col string, priority, order int) {
	if !safeIdentifier.MatchString(col) {
		log.Fatalf("column name %q in index %q contains unsafe characters", col, b.name)
	}
	b.columns = append(b.columns, indexColumn{name: col, priority: priority, order: order})
}

func (b *indexBuilder) build() IndexInfo {
	sorted := make([]indexColumn, len(b.columns))
	copy(sorted, b.columns)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].priority != sorted[j].priority {
			return sorted[i].priority < sorted[j].priority
		}
		return sorted[i].order < sorted[j].order
	})
	names := make([]string, 0, len(sorted))
	for _, c := range sorted {
		names = append(names, c.name)
	}
	cols := strings.Join(names, ", ")
	createSQL := fmt.Sprintf("CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING %s (%s)", b.name, b.table, b.indexType, cols)

	return IndexInfo{
		Name:      b.name,
		CreateSQL: createSQL,
	}
}

// collectIndexes extracts all index definitions from a schema, grouping fields by index name
// to handle composite indexes. It also generates SAC filter indexes for tables with
// ClusterID or Namespace search fields.
func collectIndexes(schema *walker.Schema, obj object) []IndexInfo {
	tablePrefix := strings.ToLower(lowerCamelCase(schema.Table))
	table := strings.ToLower(schema.Table)
	if !safeIdentifier.MatchString(table) {
		log.Fatalf("table name %q contains unsafe characters", table)
	}

	idxNameToBuilder := make(map[string]*indexBuilder)
	// idxBuildOrder is used to consistently iterator of idxNameToBuilder
	var idxBuildOrder []string
	// fieldOrder is a monotonic counter over proto field order. It is the tiebreaker
	// used to keep composite-index columns in field order when they share a priority
	// (including the no-priority sentinel), so output stays byte-identical today.
	fieldOrder := 0
	for _, field := range schema.DBColumnFields() {
		col := strings.ToLower(field.ColumnName)
		order := fieldOrder
		fieldOrder++

		for _, idx := range field.Options.Index {
			if idx.IndexCategory == "unique" {
				continue
			}
			name := idx.IndexName
			if name == "" {
				name = tablePrefix + "_" + col
			}
			if !safeIdentifier.MatchString(name) {
				log.Fatalf("index name %q contains unsafe characters — must match [a-z_][a-z0-9_]*", name)
			}

			prio := noIndexPriority
			if idx.IndexPriority != "" {
				p, err := strconv.Atoi(idx.IndexPriority)
				if err != nil {
					log.Fatalf("index %q: non-integer priority %q", name, idx.IndexPriority)
				}
				prio = p
			}

			if b, ok := idxNameToBuilder[name]; ok {
				b.addColumn(col, prio, order)
			} else {
				indexType := idx.IndexType
				if indexType == "" {
					indexType = "btree"
				}
				if !safeIdentifier.MatchString(indexType) {
					log.Fatalf("index type %q in index %q contains unsafe characters", indexType, name)
				}
				b = &indexBuilder{
					name:      name,
					table:     table,
					indexType: indexType,
				}
				b.addColumn(col, prio, order)
				idxNameToBuilder[name] = b
				idxBuildOrder = append(idxBuildOrder, name)
			}
		}
	}

	if sacBuilder := buildSACFilterIndex(schema, obj, tablePrefix, table); sacBuilder != nil {
		idxNameToBuilder[sacBuilder.name] = sacBuilder
		idxBuildOrder = append(idxBuildOrder, sacBuilder.name)
	}

	result := make([]IndexInfo, 0, len(idxBuildOrder))
	for _, name := range idxBuildOrder {
		result = append(result, idxNameToBuilder[name].build())
	}
	return result
}

// buildSACFilterIndex creates a composite index on ClusterID/Namespace fields for
// scope-based access control filtering. Returns nil if no SAC fields are present.
func buildSACFilterIndex(schema *walker.Schema, obj object, tablePrefix, table string) *indexBuilder {
	b := &indexBuilder{
		name:      tablePrefix + "_sac_filter",
		table:     table,
		indexType: "btree",
	}
	if obj.IsClusterScope() {
		b.indexType = "hash"
	}

	order := 0
	for _, field := range schema.DBColumnFields() {
		if field.Options.PrimaryKey {
			continue
		}
		if field.Search.FieldName == search.ClusterID.String() || field.Search.FieldName == search.Namespace.String() {
			b.addColumn(strings.ToLower(field.ColumnName), noIndexPriority, order)
			order++
		}
	}

	if len(b.columns) == 0 {
		return nil
	}
	return b
}

var funcMap = template.FuncMap{
	"arr":                          arr,
	"lowerCamelCase":               lowerCamelCase,
	"upperCamelCase":               upperCamelCase,
	"valueExpansion":               valueExpansion,
	"lowerCase":                    strings.ToLower,
	"concatWith":                   concatWith,
	"searchFieldNameInOtherSchema": searchFieldNameInOtherSchema,
	"isSacScoping":                 isSacScoping,
	"isMessageBytes":               isMessageBytes,
	"messageBytesElemType":         messageBytesElemType,
	"subMessageInits":              subMessageInits,
	"trimPrefix":                   func(prefix, s string) string { return strings.TrimPrefix(s, prefix) },
	"collectIndexes":               collectIndexes,
	"dict":                         dict,
	"pluralType": func(s string) string {
		if s[len(s)-1] == 'y' {
			return fmt.Sprintf("%sies", strings.TrimSuffix(s, "y"))
		}
		return fmt.Sprintf("%ss", s)
	},
	"objectGetterToSetter": func(getter string) string {
		s := strings.TrimSuffix(getter, "()")
		s = strings.TrimPrefix(s, "Get")
		return s
	},
}
