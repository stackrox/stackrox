package main

import (
	"bytes"
	"io/ioutil"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	scrapeFixturePath  = "/files/scrape.json"
	resultsFixturePath = "/files/results.json"
)

var (
	defaultComplianceReturn *compliance.ComplianceReturn
	defaultCheckResults     *compliance.ComplianceReturn
)

func init() {
	defaultComplianceReturn = loadComplianceReturn(scrapeFixturePath)
	defaultCheckResults = loadComplianceReturn(resultsFixturePath)
}

func loadComplianceReturn(path string) *compliance.ComplianceReturn {
	complianceBytes, err := ioutil.ReadFile(path)
	utils.Must(err)

	buf := bytes.NewBuffer(complianceBytes)
	var complianceReturn compliance.ComplianceReturn
	utils.Must(jsonpb.Unmarshal(buf, &complianceReturn))
	return &complianceReturn
}

func getComplianceReturn(scrapeID, nodeName string) *compliance.ComplianceReturn {
	cr := defaultComplianceReturn.Clone()
	cr.ScrapeId = scrapeID
	cr.NodeName = nodeName
	return cr
}

func getCheckResults(scrapeID, nodeName string) *compliance.ComplianceReturn {
	cr := defaultCheckResults.Clone()
	cr.ScrapeId = scrapeID
	cr.NodeName = nodeName
	cr.Time = types.TimestampNow()
	return cr
}
