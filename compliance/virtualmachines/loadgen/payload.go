package main

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"google.golang.org/protobuf/proto"
)

// payloadProvider provides pre-generated and pre-marshaled payloads for each CID.
type payloadProvider struct {
	payloads map[uint32][]byte
}

func newPayloadProvider(generator *vmindexreport.Generator, vmCount int, startCID uint32) (*payloadProvider, error) {
	endCID := startCID + uint32(vmCount) - 1

	log.Infof("pre-generating %d unique reports for CID range [%d-%d]...", vmCount, startCID, endCID)
	start := time.Now()

	payloads := make(map[uint32][]byte)

	for i := 0; i < vmCount; i++ {
		cid := startCID + uint32(i)
		report := generator.GenerateV1IndexReport(cid)

		data, err := proto.Marshal(report)
		if err != nil {
			return nil, fmt.Errorf("marshal report for CID %d: %w", cid, err)
		}
		payloads[cid] = data
	}

	log.Infof("pre-generated %d unique reports in %s", len(payloads), time.Since(start))
	return &payloadProvider{payloads: payloads}, nil
}

func (p *payloadProvider) get(cid uint32) ([]byte, error) {
	payload, ok := p.payloads[cid]
	if !ok {
		return nil, fmt.Errorf("CID %d not in pre-generated range", cid)
	}
	return payload, nil
}
