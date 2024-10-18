package sachelper

import (
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

func extractScopeElements(results []search.Result, optionsMap search.OptionsMap, searchedField string) []*v1.ScopeObject {
	scopeElements := make([]*v1.ScopeObject, 0, len(results))
	targetField, fieldFound := optionsMap.Get(searchedField)
	for _, r := range results {
		objID := r.ID
		objName := ""
		if fieldFound {
			for _, v := range r.Matches[targetField.GetFieldPath()] {
				if len(v) > 0 {
					objName = v
					break
				}
			}
		}
		if len(objName) == 0 {
			objName = fmt.Sprintf("%s with ID %s", searchedField, objID)
		}
		element := &v1.ScopeObject{
			Id:   objID,
			Name: objName,
		}
		scopeElements = append(scopeElements, element)
	}
	return scopeElements
}
