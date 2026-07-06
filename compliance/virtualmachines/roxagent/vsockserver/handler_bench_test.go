package vsockserver

import (
	"net"
	"os"
	"testing"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

// TestMain raises the log level before benchmarking: Handler.HandleConn logs
// one line per request at Info level, and at these iteration counts (tens of
// thousands per second for the cheap "unchanged" path) that logging call is
// not free -- left at the default level it would measure stdout I/O instead
// of roxagent's own request-handling cost.
func TestMain(m *testing.M) {
	logging.SetGlobalLogLevel(zapcore.ErrorLevel)
	os.Exit(m.Run())
}

// Part B of the VSOCK pull-mode stress-test tooling (see
// docs/superpowers/specs/2026-07-03-vsock-pull-stress-test-design.md):
// measures roxagent's own per-request cost in isolation -- frame read,
// protobuf unmarshal, generation compare, protobuf marshal, frame write --
// via the real Handler.HandleConn over net.Pipe. No VSOCK/TLS/dial overhead
// and no Sensor-side concurrency are involved, so this isolates whether
// roxagent's own handling is ever the bottleneck (part of Q3), distinct from
// Part A's end-to-end (Sensor + protocol) numbers.
const benchMaxResponseSize = 16 << 20

// referencePackageCount matches the ~450KB/524-package fixture shape used
// throughout Part A, for direct comparison against those results.
const referencePackageCount = 524

func BenchmarkHandleConn_FullReport(b *testing.B) {
	handler := newBenchHandler(b, referencePackageCount)
	reqData := marshalGetReportRequest(b, 0) // below current generation (1) -> always the full-report path
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		playRequest(b, handler, reqData)
	}
}

// BenchmarkHandleConn_Unchanged measures the cheap "unchanged" ack path,
// which VMScraper takes on every poll after the first once a VM's report has
// stopped changing -- the common case in a stable fleet, and much of what
// Part A's default (non-always-changed) runs actually exercised.
func BenchmarkHandleConn_Unchanged(b *testing.B) {
	handler := newBenchHandler(b, referencePackageCount)
	reqData := marshalGetReportRequest(b, 1) // matches newBenchHandler's single SetReport call (generation 1)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		playRequest(b, handler, reqData)
	}
}

// BenchmarkHandleConn_FullReport_Small and _Large sweep report size to show
// how roxagent's own cost scales with report size -- customer VM fleets are
// unlikely to be as uniform as the ~524-package reference fixture.
func BenchmarkHandleConn_FullReport_Small(b *testing.B) {
	handler := newBenchHandler(b, 50)
	reqData := marshalGetReportRequest(b, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		playRequest(b, handler, reqData)
	}
}

func BenchmarkHandleConn_FullReport_Large(b *testing.B) {
	handler := newBenchHandler(b, 2000)
	reqData := marshalGetReportRequest(b, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		playRequest(b, handler, reqData)
	}
}

func newBenchHandler(b *testing.B, numPackages int) *Handler {
	b.Helper()
	gen := vmindexreport.NewGeneratorWithSeed(numPackages, 42)
	report := gen.GenerateV4IndexReport()
	cache := &ReportCache{}
	cache.SetReport(report, map[string]string{"detected_os": "RHEL", "os_version": "9.4"})
	return NewHandler(cache, "bench-agent")
}

func marshalGetReportRequest(b *testing.B, ifNewerThan uint32) []byte {
	b.Helper()
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "bench"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{IfNewerThanGeneration: ifNewerThan}},
	}
	data, err := proto.Marshal(req)
	require.NoError(b, err)
	return data
}

// playRequest drives one real HandleConn round trip over net.Pipe -- the same
// transport-free wiring Part A's FarmDialer uses -- so this measures purely
// roxagent's own request-handling cost, with zero dial/network cost on
// either side.
func playRequest(b *testing.B, handler *Handler, reqData []byte) {
	b.Helper()
	clientConn, serverConn := net.Pipe()
	go handler.HandleConn(serverConn)

	require.NoError(b, vsockframing.WriteFrame(clientConn, reqData))
	_, err := vsockframing.ReadFrame(clientConn, benchMaxResponseSize)
	require.NoError(b, err)
	_ = clientConn.Close()
}
