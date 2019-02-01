package handler

import "testing"

func TestGetOptions(t *testing.T) {
	mocks := mockResolver(t)
	res := executeTestQuery(t, mocks, "{searchOptions()}")
	assertNoErrors(t, res.Body)
	assertJSONMatches(t, res.Body, ".data.searchOptions[0]", "Add Capabilities")
	res = executeTestQuery(t, mocks, "{searchOptions(categories: [DEPLOYMENTS])}")
	assertNoErrors(t, res.Body)
	//t.Log(res.Body)
}
