package main

import (
	"fmt"
	"log"
	"strings"
	"text/template"
	"unicode"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/stringutils"
)

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

var funcMap = template.FuncMap{
	"lowerCamelCase": lowerCamelCase,
	"upperCamelCase": upperCamelCase,
	"valueExpansion": valueExpansion,
	"lowerCase":      strings.ToLower,
}

func compileFKArgsForSchema(schema *walker.Schema, refArgs []string) []*walker.ReferenceInfo {
	var refs []*walker.ReferenceInfo
	for _, refArg := range refArgs {
		tablePart := refArg[:strings.Index(refArg, "(")]
		refTable, refObjType := stringutils.Split2(tablePart, ":")
		refMsgType := proto.MessageType(refObjType)
		if refMsgType == nil {
			log.Fatalf("could not find message for type: %s", refObjType)
		}
		pSchema := walker.Walk(refMsgType, refTable)

		keyPart := refArg[strings.Index(refArg, "(")+1 : strings.Index(refArg, ")")]
		for _, part := range strings.Split(keyPart, ";") {
			fk, refField := stringutils.Split2(part, ":")
			ref := &walker.ReferenceInfo{
				ForeignKey: fk,
				RefSchema:  pSchema,
				Reference:  refField,
			}
			schema = schema.WithReference(ref)
			refs = append(refs, ref)
		}
	}
	return refs
}
