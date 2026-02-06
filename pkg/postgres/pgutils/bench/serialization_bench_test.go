//go:build !sql_integration

package bench

import (
	"runtime"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// BenchmarkMarshalVT benchmarks VTProtobuf binary marshaling.
func BenchmarkMarshalVT(b *testing.B) {
	deployment := fixtures.GetDeployment()
	image := fixtures.GetImage()
	largeImage := fixtures.GetImageWithUniqueComponents(50)

	b.Run("Deployment", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := deployment.MarshalVT(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Image", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := image.MarshalVT(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeImage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := largeImage.MarshalVT(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMarshalProtojson benchmarks protojson JSON marshaling.
func BenchmarkMarshalProtojson(b *testing.B) {
	deployment := fixtures.GetDeployment()
	image := fixtures.GetImage()
	largeImage := fixtures.GetImageWithUniqueComponents(50)

	b.Run("Deployment", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := protojson.Marshal(deployment); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Image", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := protojson.Marshal(image); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeImage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := protojson.Marshal(largeImage); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkUnmarshalVTUnsafe benchmarks VTProtobuf binary unmarshaling.
func BenchmarkUnmarshalVTUnsafe(b *testing.B) {
	deployment := fixtures.GetDeployment()
	image := fixtures.GetImage()
	largeImage := fixtures.GetImageWithUniqueComponents(50)

	deploymentBytes, _ := deployment.MarshalVT()
	imageBytes, _ := image.MarshalVT()
	largeImageBytes, _ := largeImage.MarshalVT()

	b.Run("Deployment", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg storage.Deployment
			if err := msg.UnmarshalVTUnsafe(deploymentBytes); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Image", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg storage.Image
			if err := msg.UnmarshalVTUnsafe(imageBytes); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeImage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg storage.Image
			if err := msg.UnmarshalVTUnsafe(largeImageBytes); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkUnmarshalProtojson benchmarks protojson JSON unmarshaling.
func BenchmarkUnmarshalProtojson(b *testing.B) {
	deployment := fixtures.GetDeployment()
	image := fixtures.GetImage()
	largeImage := fixtures.GetImageWithUniqueComponents(50)

	deploymentJSON, _ := protojson.Marshal(deployment)
	imageJSON, _ := protojson.Marshal(image)
	largeImageJSON, _ := protojson.Marshal(largeImage)

	b.Run("Deployment", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg storage.Deployment
			if err := protojson.Unmarshal(deploymentJSON, &msg); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Image", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg storage.Image
			if err := protojson.Unmarshal(imageJSON, &msg); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeImage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg storage.Image
			if err := protojson.Unmarshal(largeImageJSON, &msg); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSerializedSize reports the byte sizes of both formats.
func BenchmarkSerializedSize(b *testing.B) {
	deployment := fixtures.GetDeployment()
	image := fixtures.GetImage()
	largeImage := fixtures.GetImageWithUniqueComponents(50)

	type testCase struct {
		name string
		msg  proto.Message
	}

	cases := []testCase{
		{"Deployment", deployment},
		{"Image", image},
		{"LargeImage", largeImage},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vtBytes, _ := proto.Marshal(tc.msg)
			jsonBytes, _ := protojson.Marshal(tc.msg)

			b.ReportMetric(float64(len(vtBytes)), "bytes/protobuf")
			b.ReportMetric(float64(len(jsonBytes)), "bytes/json")
			if len(vtBytes) > 0 {
				b.ReportMetric(float64(len(jsonBytes))/float64(len(vtBytes)), "json/proto-ratio")
			}

			// Run a trivial loop to satisfy the benchmark framework.
			for i := 0; i < b.N; i++ {
			}
		})
	}
}

// BenchmarkMemoryBytea measures total heap allocations for 1000 unmarshal operations (bytea/protobuf).
func BenchmarkMemoryBytea(b *testing.B) {
	deployment := fixtures.GetDeployment()
	data, _ := deployment.MarshalVT()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var memBefore, memAfter runtime.MemStats
		runtime.ReadMemStats(&memBefore)
		for j := 0; j < 1000; j++ {
			var msg storage.Deployment
			if err := msg.UnmarshalVTUnsafe(data); err != nil {
				b.Fatal(err)
			}
		}
		runtime.ReadMemStats(&memAfter)
		b.ReportMetric(float64(memAfter.TotalAlloc-memBefore.TotalAlloc), "heap-bytes/1000ops")
	}
}

// BenchmarkMemoryJsonb measures total heap allocations for 1000 unmarshal operations (jsonb/protojson).
func BenchmarkMemoryJsonb(b *testing.B) {
	deployment := fixtures.GetDeployment()
	data, _ := protojson.Marshal(deployment)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var memBefore, memAfter runtime.MemStats
		runtime.ReadMemStats(&memBefore)
		for j := 0; j < 1000; j++ {
			var msg storage.Deployment
			if err := protojson.Unmarshal(data, &msg); err != nil {
				b.Fatal(err)
			}
		}
		runtime.ReadMemStats(&memAfter)
		b.ReportMetric(float64(memAfter.TotalAlloc-memBefore.TotalAlloc), "heap-bytes/1000ops")
	}
}
