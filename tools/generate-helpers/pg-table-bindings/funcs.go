package main

import (
	"fmt"
	"log"
	"strings"
	"text/template"
	"unicode"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
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

		refMsgType := proto.MessageType(refObjType)
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
	applyPointwise(words[1:], strings.Title)
	return strings.Join(words, "")
}

func upperCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	applyPointwise(words, strings.Title)
	return strings.Join(words, "")
}

func valueExpansion(size int) string {
	var all []string
	for i := 0; i < size; i++ {
		all = append(all, fmt.Sprintf("$%d", i+1))
	}
	return strings.Join(all, ", ")
}

func concatWith(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

var funcMap = template.FuncMap{
	"lowerCamelCase":               lowerCamelCase,
	"upperCamelCase":               upperCamelCase,
	"valueExpansion":               valueExpansion,
	"lowerCase":                    strings.ToLower,
	"storageToResource":            storageToResource,
	"concatWith":                   concatWith,
	"searchFieldNameInOtherSchema": searchFieldNameInOtherSchema,
	"pluralType": func(s string) string {
		if s[len(s)-1] == 'y' {
			return fmt.Sprintf("%sies", strings.TrimSuffix(s, "y"))
		}
		return fmt.Sprintf("%ss", s)
	},
}
