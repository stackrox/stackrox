package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	for _, records := range [][]proto.Message{
		{exportFixture(1), exportFixture(2)},
		{&v1.ExportDeploymentResponse{}, &v1.ExportDeploymentResponse{Deployment: &storage.Deployment{Name: "a&b"}}},
		{&v1.ExportNodeResponse{}, &v1.ExportNodeResponse{Node: &storage.Node{Name: "a&b"}}},
		{&v1.ExportPodResponse{}, &v1.ExportPodResponse{Pod: &storage.Pod{Name: "a&b"}}},
		{&v1.VulnMgmtExportWorkloadsResponse{}, &v1.VulnMgmtExportWorkloadsResponse{Images: []*storage.Image{exportFixture(1).GetImage()}}},
	} {
		var baseline []byte
		for _, name := range []string{"baseline", "direct"} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			forwardExport(runtime.NewServerMetadataContext(context.Background(), runtime.ServerMetadata{}), runtime.NewServeMux(), variants[name], w, req, records, status.Error(codes.Internal, "test error"))
			lines := bytes.Split(bytes.TrimSpace(w.Body.Bytes()), []byte("\n"))
			require.Len(t, lines, 3)
			require.Contains(t, string(lines[2]), "test error")
			if name == "baseline" {
				baseline = w.Body.Bytes()
			} else {
				expected := bytes.Split(bytes.TrimSpace(baseline), []byte("\n"))
				for i := range lines {
					require.JSONEq(t, string(expected[i]), string(lines[i]))
				}
			}
		}
	}
}

type exportWriter struct{ http.ResponseWriter }

func (exportWriter) Write(p []byte) (int, error) { return len(p), nil }
func (exportWriter) Flush()                      {}

// ForwardResponseStream writes each compact chunk and its newline before flushing.
type exportPreflightWriter struct {
	http.ResponseWriter
	chunk   bytes.Buffer
	results int
	err     error
}

func (w *exportPreflightWriter) Write(p []byte) (int, error) { return w.chunk.Write(p) }
func (w *exportPreflightWriter) Flush() {
	if w.err != nil {
		return
	}
	var envelope struct{ Result json.RawMessage }
	if err := json.Unmarshal(w.chunk.Bytes(), &envelope); err != nil {
		w.err = err
	} else if len(envelope.Result) == 0 {
		w.err = fmt.Errorf("expected result chunk at record %d", w.results+1)
	} else {
		w.results++
	}
	w.chunk.Reset()
}

func BenchmarkImageExportMarshaler(b *testing.B) {
	variants := exportVariants()
	for _, n := range []int{0, 32, 256} {
		image := exportFixture(n)
		env := map[string]any{"result": image}
		for name, m := range variants {
			b.Run(fmt.Sprintf("components=%d/encoder=%s", n, name), func(b *testing.B) {
				buf, err := m.Marshal(env)
				require.NoError(b, err)
				b.ReportAllocs()
				for b.Loop() {
					if _, err := m.Marshal(env); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(len(buf)), "encoded-B/image")
			})
		}
	}
}

func BenchmarkExportStream(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		run := func(name string, build func(int) proto.Message) {
			b.Run(fmt.Sprintf("stream=%d/%s", size, name), func(b *testing.B) {
				records := make([]proto.Message, size)
				for i := range records {
					records[i] = build(i)
				}
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := runtime.NewServerMetadataContext(context.Background(), runtime.ServerMetadata{})
				mux := runtime.NewServeMux()
				for name, m := range exportVariants() {
					b.Run("encoder="+name, func(b *testing.B) {
						preflight := &exportPreflightWriter{ResponseWriter: httptest.NewRecorder()}
						forwardExport(ctx, mux, m, preflight, req, records, io.EOF)
						require.NoError(b, preflight.err)
						require.Equal(b, len(records), preflight.results)
						require.Zero(b, preflight.chunk.Len())
						w := exportWriter{httptest.NewRecorder()}
						b.ReportAllocs()
						sample, err := m.Marshal(map[string]any{"result": records[0]})
						require.NoError(b, err)
						for b.Loop() {
							forwardExport(ctx, mux, m, w, req, records, io.EOF)
						}
						b.ReportMetric(float64(len(sample)+1), "wire-B/record")
					})
				}
			})
		}
		for _, n := range []int{0, 32, 256} {
			scan := exportFixture(n).GetImage().GetScan()
			image := func(i int) *storage.Image { return &storage.Image{Id: fmt.Sprintf("sha256:%064x", i), Scan: scan} }
			run(fmt.Sprintf("image/components=%d", n), func(i int) proto.Message { return &v1.ExportImageResponse{Image: image(i)} })
			for _, count := range []int{1, 4} {
				run(fmt.Sprintf("workload/images=%d/components=%d", count, n), func(i int) proto.Message {
					response := &v1.VulnMgmtExportWorkloadsResponse{Deployment: &storage.Deployment{Id: fmt.Sprintf("%08d", i)}, LivePods: 2}
					for j := range count {
						response.Images = append(response.Images, image(i*count+j))
					}
					return response
				})
			}
		}
		for _, n := range []int{1, 4, 16} {
			run(fmt.Sprintf("deployment/containers=%d", n), func(i int) proto.Message {
				d := &storage.Deployment{Id: fmt.Sprintf("%08d", i), Namespace: "production"}
				for j := range n {
					d.Containers = append(d.Containers, &storage.Container{Name: fmt.Sprintf("container-%d", j)})
				}
				return &v1.ExportDeploymentResponse{Deployment: d}
			})
			run(fmt.Sprintf("pod/instances=%d", n), func(i int) proto.Message {
				p := &storage.Pod{Id: fmt.Sprintf("%08d", i), Namespace: "production"}
				for j := range n {
					p.LiveInstances = append(p.LiveInstances, &storage.ContainerInstance{ContainerName: fmt.Sprintf("container-%d", j), ContainingPodId: p.GetId()})
				}
				return &v1.ExportPodResponse{Pod: p}
			})
		}
		for _, n := range []int{0, 16, 64} {
			run(fmt.Sprintf("node/labels=%d", n), func(i int) proto.Message {
				node := &storage.Node{Id: fmt.Sprintf("%08d", i), ClusterName: "production", ContainerRuntimeVersion: "containerd://2.0", Labels: map[string]string{}}
				for j := range n {
					node.Labels[fmt.Sprintf("label-%02d", j)] = "node-label-value"
				}
				return &v1.ExportNodeResponse{Node: node}
			})
		}
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
