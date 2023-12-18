package proto_bench

import (
	"testing"

	cs "github.com/stackrox/stackrox/proto_bench/csproto/generated/storage"
	gogo "github.com/stackrox/stackrox/proto_bench/gogo/generated/storage"
	vt "github.com/stackrox/stackrox/proto_bench/vtproto/generated/storage"
)

func BenchmarkMarshal(b *testing.B) {
	var _ gogo.Cluster
	var _ vt.Cluster
	var _ cs.Cluster
}
