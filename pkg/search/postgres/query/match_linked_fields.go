package pgsearch

//func newMapSearchField(path string, sf *searchPkg.Field) *searchPkg.Field {
//	return &searchPkg.Field{
//		FieldPath: path,
//		Type:      v1.SearchDataType_SEARCH_STRING,
//		Store:     sf.GetStore(),
//		Hidden:    sf.GetHidden(),
//		Category:  sf.Category,
//	}
//}
//
//func getMapSearchFieldsAndValues(key, value string, fv searchFieldAndValue, highlightCtx highlightContext) (searchFieldAndValue, searchFieldAndValue) {
//	if key == "" || key == searchPkg.WildcardString {
//		key = searchPkg.RegexQueryString(".*")
//	}
//	if value == "" {
//		value = searchPkg.WildcardString
//	}
//	keySearchField := newMapSearchField(ToMapKeyPath(fv.sf.GetFieldPath()), fv.sf)
//	keyFv := searchFieldAndValue{sf: keySearchField, value: strings.TrimPrefix(key, searchPkg.NegationPrefix), highlight: fv.highlight && highlightCtx != nil}
//
//	valueSearchField := newMapSearchField(ToMapValuePath(fv.sf.GetFieldPath()), fv.sf)
//	valueFv := searchFieldAndValue{sf: valueSearchField, value: strings.TrimPrefix(value, searchPkg.NegationPrefix), highlight: fv.highlight && highlightCtx != nil}
//	return keyFv, valueFv
//}
//
//func handleNegatedMapQuery(ctx bleveContext, index bleve.Index, category v1.SearchCategory, fv searchFieldAndValue, highlightCtx highlightContext) (query.Query, error) {
//	key, value := parseLabel(fv.value)
//	keyFv, valueFv := getMapSearchFieldsAndValues(key, value, fv, highlightCtx)
//
//	docIDQuery, err := matchAllFieldsQuery(ctx, index, category, []searchFieldAndValue{keyFv, valueFv}, highlightCtx)
//	if err != nil {
//		return nil, err
//	}
//	bq := newBooleanQuery(category)
//	bq.AddMustNot(docIDQuery)
//	switch {
//	case searchPkg.IsNegationQuery(key) && searchPkg.IsNegationQuery(value):
//		// We just need the boolean query with must not
//	case searchPkg.IsNegationQuery(key):
//		// Add must match value
//		mfq, err := matchFieldQuery(category, valueFv.sf.GetFieldPath(), valueFv.sf.GetType(), valueFv.value)
//		if err != nil {
//			return nil, err
//		}
//		bq.AddMust(mfq)
//		// If there is no value and !key then we should check that the key doesn't exist as well
//		if value == "" {
//			return bleve.NewDisjunctionQuery(bq, nullQuery(category, keyFv.sf.GetFieldPath())), nil
//		}
//	case searchPkg.IsNegationQuery(value):
//		// Add must match key
//		mfq, err := matchFieldQuery(category, keyFv.sf.GetFieldPath(), keyFv.sf.GetType(), keyFv.value)
//		if err != nil {
//			return nil, err
//		}
//		bq.AddMust(mfq)
//	}
//	return bq, nil
//}
//
//func matchAllFieldsQuery(ctx bleveContext, index bleve.Index, category v1.SearchCategory, fieldsAndValues []searchFieldAndValue, highlightCtx highlightContext) (query.Query, error) {
//	if len(fieldsAndValues) == 0 {
//		return bleve.NewMatchNoneQuery(), nil
//	}
//
//	filteredFieldsAndValues := fieldsAndValues[:0]
//	var mapQueries []query.Query
//	for _, fv := range fieldsAndValues {
//		if fv.sf.GetType() != v1.SearchDataType_SEARCH_MAP {
//			filteredFieldsAndValues = append(filteredFieldsAndValues, fv)
//			continue
//		}
//		key, value := parseLabel(fv.value)
//		// If we have a negation query, use the special handling. Otherwise, add them to the normal linked query
//		// This is useful for autocomplete
//		if searchPkg.IsNegationQuery(key) || searchPkg.IsNegationQuery(value) {
//			mapQuery, err := handleNegatedMapQuery(ctx, index, category, fv, highlightCtx)
//			if err != nil {
//				return nil, err
//			}
//			mapQueries = append(mapQueries, mapQuery)
//			continue
//		}
//		keyFv, valueFv := getMapSearchFieldsAndValues(key, value, fv, highlightCtx)
//		filteredFieldsAndValues = append(filteredFieldsAndValues, keyFv, valueFv)
//	}
//	fieldsAndValues = filteredFieldsAndValues
//
//	// If there's only one field, just return a "regular" search query.
//	if len(fieldsAndValues) == 1 {
//		if highlightCtx != nil {
//			highlightCtx.AddFieldToHighlight(fieldsAndValues[0].sf.GetFieldPath())
//		}
//		return matchFieldQuery(category, fieldsAndValues[0].sf.GetFieldPath(), fieldsAndValues[0].sf.GetType(), fieldsAndValues[0].value)
//	}
//
//	// If we have to match multiple fields, and check that the matches are in the corresponding positions,
//	// we perform the query, and filter the results by those which have matches in corresponding positions of different
//	// fields, and return a docID query for those fields.
//	// See the comments on tree.Tree for details on how the array positions checks work.
//	var mfQs []query.Query
//	for _, fieldAndValue := range fieldsAndValues {
//		// Wildcards have no use case except to highlight fields
//		if fieldAndValue.value == searchPkg.WildcardString {
//			if highlightCtx != nil {
//				highlightCtx.AddFieldToHighlight(fieldAndValue.sf.GetFieldPath())
//			}
//			continue
//		}
//		mfQ, err := matchFieldQuery(category, fieldAndValue.sf.GetFieldPath(), fieldAndValue.sf.GetType(), fieldAndValue.value)
//		if err != nil {
//			return nil, errors.Wrapf(err, "computing match field query for %+v (sf: %+v)", fieldAndValue, fieldAndValue.sf)
//		}
//		mfQs = append(mfQs, mfQ)
//		if fieldAndValue.highlight && highlightCtx != nil {
//			highlightCtx.AddFieldToHighlight(fieldAndValue.sf.GetFieldPath())
//		}
//	}
//	conjunction := bleve.NewConjunctionQuery(mfQs...)
//	conjunction.AddQuery(mapQueries...)
//	searchResult, err := runBleveQuery(ctx, conjunction, index, highlightCtx, true)
//	if err != nil {
//		return nil, errors.Wrapf(err, "running sub query for category %s, fieldsAndValues: %+v", category, fieldsAndValues)
//	}
//
//	var resultIDs []string
//	for _, hit := range searchResult.Hits {
//		if matched := matchAndHighlight(hit, fieldsAndValues, highlightCtx); matched {
//			resultIDs = append(resultIDs, hit.ID)
//		}
//	}
//	if len(resultIDs) == 0 {
//		return bleve.NewMatchNoneQuery(), nil
//	}
//	return bleve.NewDocIDQuery(resultIDs), nil
//}
