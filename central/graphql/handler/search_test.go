package handler

import "testing"

func TestGetOptions(t *testing.T) {
	mocks := mockResolver(t)
	res := executeTestQuery(t, mocks, "{searchOptions(categories: [])}")
	assertNoErrors(t, res.Body)
	assertJSONMatches(t, res.Body, ".data.searchOptions[0]", "Cluster")
	res = executeTestQuery(t, mocks, "{searchOptions(categories: [DEPLOYMENTS])}")
	assertNoErrors(t, res.Body)
	//t.Log(res.Body)
}
