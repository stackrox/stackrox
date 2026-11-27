package grpc

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestImageExportMarshaler(t *testing.T) {
	variants := exportVariants()
	for _, n := range []int{0, 32, 256} {
		envelope := map[string]any{"result": exportFixture(n)}
		want, err := variants["baseline"].Marshal(envelope)
		require.NoError(t, err)
		got, err := variants["direct"].Marshal(envelope)
		require.NoError(t, err)
		require.JSONEq(t, string(want), string(got))
	}
}

func TestImageExportMarshalerRejectsInvalidUTF8(t *testing.T) {
	image := &v1.ExportImageResponse{Image: &storage.Image{Id: "\xff"}}
	for name, marshaler := range exportVariants() {
		t.Run(name, func(t *testing.T) {
			_, err := marshaler.Marshal(map[string]any{"result": image})
			require.ErrorContains(t, err, "invalid UTF-8")
		})
	}
}

func TestExportStreamMatchesBaseline(t *testing.T) {
	variants := exportVariants()
	for name, records := range map[string][]proto.Message{
		"image":      {exportFixture(1), exportFixture(2)},
		"deployment": {&v1.ExportDeploymentResponse{}, &v1.ExportDeploymentResponse{Deployment: &storage.Deployment{Name: "a&b"}}},
		"node":       {&v1.ExportNodeResponse{}, &v1.ExportNodeResponse{Node: &storage.Node{Name: "a&b"}}},
		"pod":        {&v1.ExportPodResponse{}, &v1.ExportPodResponse{Pod: &storage.Pod{Name: "a&b"}}},
		"workload":   {&v1.VulnMgmtExportWorkloadsResponse{}, &v1.VulnMgmtExportWorkloadsResponse{Images: []*storage.Image{exportFixture(1).GetImage()}}},
	} {
		t.Run(name, func(t *testing.T) {
			stream := func(m runtime.Marshaler) [][]byte {
				t.Helper()
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := runtime.NewServerMetadataContext(t.Context(), runtime.ServerMetadata{})
				forwardExport(ctx, runtime.NewServeMux(), m, w, req, records, status.Error(codes.Internal, "test error"))
				lines := bytes.Split(bytes.TrimSpace(w.Body.Bytes()), []byte("\n"))
				require.Len(t, lines, len(records)+1)
				require.Contains(t, string(lines[len(records)]), "test error")
				return lines
			}
			want := stream(variants["baseline"])
			got := stream(variants["direct"])
			for i := range want {
				require.JSONEq(t, string(want[i]), string(got[i]))
			}
		})
	}
}

func TestImageExportMarshalerFallback(t *testing.T) {
	for name, tc := range map[string]struct {
		indent string
		value  any
	}{
		"pretty":         {"  ", map[string]any{"result": exportFixture(1)}},
		"other message":  {"", map[string]any{"result": &storage.Image{}}},
		"empty response": {"", map[string]any{"result": (*v1.ExportImageResponse)(nil)}},
		"extra key":      {"", map[string]any{"result": exportFixture(1), "extra": true}},
	} {
		t.Run(name, func(t *testing.T) {
			base := &runtime.JSONPb{MarshalOptions: protojson.MarshalOptions{Indent: tc.indent, EmitUnpopulated: true}}
			want, err := base.Marshal(tc.value)
			require.NoError(t, err)
			got, err := (exportMarshaler{base}).Marshal(tc.value)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func exportVariants() map[string]runtime.Marshaler {
	base := &runtime.JSONPb{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true}}
	return map[string]runtime.Marshaler{"baseline": base, "direct": exportMarshaler{base}}
}

func exportFixture(n int) *v1.ExportImageResponse {
	scan := &storage.ImageScan{OperatingSystem: "rhel:9"}
	for i := range n {
		c := &storage.EmbeddedImageScanComponent{Name: fmt.Sprintf("package-%d", i), Version: "1.2.3"}
		for j := range 3 {
			c.Vulns = append(c.Vulns, &storage.EmbeddedVulnerability{Cve: fmt.Sprintf("CVE-2026-%04d", 1000+j), Summary: "issue <tag> & > \u2028\u2029 \"quoted\" \\ path", Cvss: 7.5})
		}
		scan.Components = append(scan.Components, c)
	}
	return &v1.ExportImageResponse{Image: &storage.Image{Scan: scan}}
}

func forwardExport(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, req *http.Request, images []proto.Message, end error) {
	count := 0
	runtime.ForwardResponseStream(ctx, mux, m, w, req, func() (proto.Message, error) {
		if count == len(images) {
			return nil, end
		}
		image := images[count]
		count++
		return image, nil
	})
}
