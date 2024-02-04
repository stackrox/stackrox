/*
 *
 * Copyright 2017 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package roundrobin defines a roundrobin balancer.
// This is essentially a copy of the official implementation[0]
// modified for our specific use case.
//
// [0] https://github.com/grpc/grpc-go/blob/v1.61.0/balancer/roundrobin/roundrobin.go
package roundrobin

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/grpc/balancer/scannerv4/roundrobin/internal/grpcrand"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
)

const (
	// Name is the name of scanner_v4_round_robin balancer.
	Name = "scanner_v4_round_robin"

	createIndexReportMethod  = "/scanner.v4.Indexer/CreateIndexReport"
	getVulnerabilitiesMethod = "/scanner.v4.Matcher/GetVulnerabilities"
)

var logger = grpclog.Component("scanner_v4_roundrobin")

// newBuilder creates a new roundrobin balancer builder.
func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, &rrPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	// Note: This *must* be called inside an `init()` function. See the docs for details.
	balancer.Register(newBuilder())
}

type rrPickerBuilder struct{}

func (*rrPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	logger.Infof("scanner_v4_roundrobinPicker: Build called with info: %v", info)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
	}
	picker := &rrPicker{
		subConns:    scs,
		subConnsLen: uint32(len(scs)),
	}
	// Start at a random index, as the same RR balancer rebuilds a new
	// picker when SubConn states change, and we don't want to apply excess
	// load to the first server in the list.
	picker.next.Store(uint32(grpcrand.Intn(len(scs))))
	return picker
}

type rrPicker struct {
	// subConns is the snapshot of the roundrobin balancer when this picker was
	// created. The slice is immutable. Each Pick() will do a round robin
	// selection from it and return the selected SubConn.
	subConns    []balancer.SubConn
	subConnsLen uint32
	next        atomic.Uint32
}

func (p *rrPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	var nextIndex uint32
	switch info.FullMethodName {
	case createIndexReportMethod, getVulnerabilitiesMethod:
		// As of writing, these are the only methods which may incur
		// non-negligible CPU load, so these are the only ones worth a round-robin.
		nextIndex = p.next.Add(1)
	default:
		// Do not bother using a different connection for other methods,
		// as (as of writing) the CPU load they incur is comparatively negligible.
		nextIndex = p.next.Load()
	}
	sc := p.subConns[nextIndex%p.subConnsLen]
	logger.Infof("scanner_v4_roundrobinPicker: Choosing subconn %v for %s", sc, info.FullMethodName)
	return balancer.PickResult{SubConn: sc}, nil
}
