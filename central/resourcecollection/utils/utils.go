package utils

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

func RuleValuesToQueryList(fieldLabel pkgSearch.FieldLabel, ruleValues []*storage.RuleValue) []*v1.Query {
	ret := make([]*v1.Query, 0, len(ruleValues))
	for _, ruleValue := range ruleValues {
		ret = append(ret, pkgSearch.NewQueryBuilder().AddRegexes(fieldLabel, ruleValue.GetValue()).ProtoQuery())
	}
	return ret
}

func EmbeddedCollectionsToIDList(embeddedList []*storage.ResourceCollection_EmbeddedResourceCollection) []string {
	ret := make([]string, 0, len(embeddedList))
	for _, embedded := range embeddedList {
		ret = append(ret, embedded.GetId())
	}
	return ret
}

func IDListToEmbeddedCollections(idList []string) []*storage.ResourceCollection_EmbeddedResourceCollection {
	ret := make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(idList))
	for _, id := range idList {
		ret = append(ret, &storage.ResourceCollection_EmbeddedResourceCollection{
			Id: id,
		})
	}
	return ret
}
