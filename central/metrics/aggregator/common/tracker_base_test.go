package common

type testFinding int

var testLabelGetters = []LazyLabel[testFinding]{
	testLabel("test"),
	testLabel("Cluster"),
	testLabel("Namespace"),
	testLabel("CVE"),
	testLabel("Severity"),
	testLabel("CVSS"),
	testLabel("IsFixable"),
}

var testLabelOrder = MakeLabelOrderMap(testLabelGetters)

func testLabel(label Label) LazyLabel[testFinding] {
	return LazyLabel[testFinding]{
		label,
		func(i *testFinding) string { return testData[*i][label] }}
}
