package main

import (
	"bytes"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/utils"
)

const scrapeFixturePath = "/files/scrape.json"

var (
	defaultComplianceReturn *compliance.ComplianceReturn
)

func init() {
	scrapeData, err := ioutil.ReadFile(scrapeFixturePath)
	utils.Must(err)

	buf := bytes.NewBuffer(scrapeData)
	var complianceReturn compliance.ComplianceReturn
	utils.Must(jsonpb.Unmarshal(buf, &complianceReturn))

	defaultComplianceReturn = &complianceReturn
}

func getComplianceReturn(scrapeID, nodeName string) *compliance.ComplianceReturn {
	cr := proto.Clone(defaultComplianceReturn).(*compliance.ComplianceReturn)
	cr.ScrapeId = scrapeID
	cr.NodeName = nodeName
	return cr
}
