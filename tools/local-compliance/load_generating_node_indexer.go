package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/compliance/node"
	"github.com/stackrox/rox/compliance/utils"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

var (
	_ node.NodeIndexer = (*LoadGeneratingNodeIndexer)(nil)
)

type LoadGeneratingNodeIndexer struct {
	generationInterval time.Duration
	initialScanDelay   time.Duration
}

func (l LoadGeneratingNodeIndexer) GetIntervals() *utils.NodeScanIntervals {
	return utils.NewNodeScanInterval(l.generationInterval, 0.0, l.initialScanDelay)
}

func (l LoadGeneratingNodeIndexer) IndexNode(_ context.Context) (*v4.IndexReport, error) {
	ir := &v4.IndexReport{
		HashId:  fmt.Sprintf("sha256:%s", strings.Repeat("a", 64)),
		Success: true,
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "0",
					Name:    "openssh-clients",
					Version: "8.7p1-38.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "openssh",
						Version: "8.7p1-38.el9",
						Kind:    "source",
						Source:  nil,
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "0",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
				{
					Id:   "1",
					Name: "cpe:/a:redhat:openshift:4.16::el9",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:a:redhat:openshift:4.16:*:el9:*:*:*:*:*",
				},
			},
		},
	}
	log.Info("Generating Node IndexReport")
	return ir, nil
}
