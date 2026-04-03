package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeTestIndicator() *ProcessIndicatorJsonb {
	return &ProcessIndicatorJsonb{
		Id:            "test-id-001",
		DeploymentId:  "deploy-id-001",
		ContainerName: "my-container",
		PodId:         "pod-id-001",
		PodUid:        "pod-uid-001",
		ClusterId:     "cluster-id-001",
		Namespace:     "default",
		ImageId:       "image-id-001",
		ContainerStartTime: &timestamppb.Timestamp{
			Seconds: 1700000000,
			Nanos:   123456789,
		},
		Signal: &ProcessSignalJsonb{
			Id:           "signal-id-001",
			ContainerId:  "container-id-001",
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "/usr/bin/apt-get",
			Pid:          1234,
			Uid:          1000,
			Gid:          500,
			Scraped:      true,
			Lineage:      []string{"/bin/bash", "/sbin/init"},
			Time: &timestamppb.Timestamp{
				Seconds: 1700000001,
				Nanos:   0,
			},
			LineageInfo: []*ProcessSignalJsonb_LineageInfoJsonb{
				{ParentUid: 22, ParentExecFilePath: "/bin/bash"},
				{ParentUid: 1, ParentExecFilePath: "/sbin/init"},
			},
		},
	}
}

func TestMarshalFastJSON_MatchesProtojson(t *testing.T) {
	msg := makeTestIndicator()

	fastData, err := msg.MarshalFastJSON()
	require.NoError(t, err)

	// Unmarshal fastjson output with protojson to verify it's valid proto JSON
	roundTrip := &ProcessIndicatorJsonb{}
	require.NoError(t, protojson.Unmarshal(fastData, roundTrip))
	assert.True(t, proto.Equal(msg, roundTrip), "protojson.Unmarshal(MarshalFastJSON) should equal original")
}

func TestUnmarshalFastJSON_FromProtojson(t *testing.T) {
	msg := makeTestIndicator()

	// Marshal with protojson (canonical)
	canonical, err := protojson.Marshal(msg)
	require.NoError(t, err)

	// Unmarshal with fastjson
	roundTrip := &ProcessIndicatorJsonb{}
	require.NoError(t, roundTrip.UnmarshalFastJSON(canonical))
	assert.True(t, proto.Equal(msg, roundTrip), "UnmarshalFastJSON(protojson.Marshal) should equal original")
}

func TestRoundTrip_FastJSON(t *testing.T) {
	msg := makeTestIndicator()

	// Marshal with fastjson, unmarshal with fastjson
	data, err := msg.MarshalFastJSON()
	require.NoError(t, err)

	roundTrip := &ProcessIndicatorJsonb{}
	require.NoError(t, roundTrip.UnmarshalFastJSON(data))
	assert.True(t, proto.Equal(msg, roundTrip), "fastjson round-trip should preserve message")
}

func TestRoundTrip_CrossProtojson(t *testing.T) {
	msg := makeTestIndicator()

	// Marshal with protojson, unmarshal with fastjson
	protoData, err := protojson.Marshal(msg)
	require.NoError(t, err)
	rt1 := &ProcessIndicatorJsonb{}
	require.NoError(t, rt1.UnmarshalFastJSON(protoData))

	// Marshal with fastjson, unmarshal with protojson
	fastData, err := msg.MarshalFastJSON()
	require.NoError(t, err)
	rt2 := &ProcessIndicatorJsonb{}
	require.NoError(t, protojson.Unmarshal(fastData, rt2))

	assert.True(t, proto.Equal(rt1, rt2), "cross-format round-trips should produce equal messages")
	assert.True(t, proto.Equal(msg, rt1), "all should equal original")
}

func TestMarshalFastJSON_EmptyMessage(t *testing.T) {
	msg := &ProcessIndicatorJsonb{}

	data, err := msg.MarshalFastJSON()
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))
}

func TestMarshalFastJSON_NilMessage(t *testing.T) {
	var msg *ProcessIndicatorJsonb
	data, err := msg.MarshalFastJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))
}

func TestMarshalFastJSON_PartialMessage(t *testing.T) {
	msg := &ProcessIndicatorJsonb{
		Id:        "only-id",
		Namespace: "only-ns",
	}

	data, err := msg.MarshalFastJSON()
	require.NoError(t, err)

	// Verify round-trip through protojson
	roundTrip := &ProcessIndicatorJsonb{}
	require.NoError(t, protojson.Unmarshal(data, roundTrip))
	assert.True(t, proto.Equal(msg, roundTrip))
}

func TestUnmarshalFastJSON_SnakeCaseFieldNames(t *testing.T) {
	// protojson parsers must accept both camelCase and snake_case
	jsonData := []byte(`{
		"id": "test-1",
		"deployment_id": "deploy-1",
		"container_name": "cname",
		"pod_id": "pod-1",
		"pod_uid": "puid-1",
		"cluster_id": "cluster-1",
		"image_id": "img-1",
		"signal": {
			"container_id": "cid-1",
			"exec_file_path": "/bin/test",
			"lineage_info": [
				{"parent_uid": 42, "parent_exec_file_path": "/bin/sh"}
			]
		}
	}`)

	msg := &ProcessIndicatorJsonb{}
	require.NoError(t, msg.UnmarshalFastJSON(jsonData))

	assert.Equal(t, "test-1", msg.Id)
	assert.Equal(t, "deploy-1", msg.DeploymentId)
	assert.Equal(t, "cname", msg.ContainerName)
	assert.Equal(t, "pod-1", msg.PodId)
	assert.Equal(t, "puid-1", msg.PodUid)
	assert.Equal(t, "cluster-1", msg.ClusterId)
	assert.Equal(t, "img-1", msg.ImageId)
	require.NotNil(t, msg.Signal)
	assert.Equal(t, "cid-1", msg.Signal.ContainerId)
	assert.Equal(t, "/bin/test", msg.Signal.ExecFilePath)
	require.Len(t, msg.Signal.LineageInfo, 1)
	assert.Equal(t, uint32(42), msg.Signal.LineageInfo[0].ParentUid)
	assert.Equal(t, "/bin/sh", msg.Signal.LineageInfo[0].ParentExecFilePath)
}

func TestUnmarshalFastJSON_UnknownFields(t *testing.T) {
	jsonData := []byte(`{"id": "test-1", "unknown_field": "should be skipped", "nested_unknown": {"a": 1}}`)

	msg := &ProcessIndicatorJsonb{}
	require.NoError(t, msg.UnmarshalFastJSON(jsonData))
	assert.Equal(t, "test-1", msg.Id)
}

func TestUnmarshalFastJSON_NullFields(t *testing.T) {
	jsonData := []byte(`{"id": "test-1", "signal": null, "containerStartTime": null}`)

	msg := &ProcessIndicatorJsonb{}
	require.NoError(t, msg.UnmarshalFastJSON(jsonData))
	assert.Equal(t, "test-1", msg.Id)
	assert.Nil(t, msg.Signal)
	assert.Nil(t, msg.ContainerStartTime)
}

func TestMarshalFastJSON_StringEscaping(t *testing.T) {
	msg := &ProcessIndicatorJsonb{
		Id:        `has "quotes" and \backslash`,
		Namespace: "has\nnewline\tand\ttabs",
	}

	data, err := msg.MarshalFastJSON()
	require.NoError(t, err)

	// Verify protojson can parse the escaped output
	roundTrip := &ProcessIndicatorJsonb{}
	require.NoError(t, protojson.Unmarshal(data, roundTrip))
	assert.Equal(t, msg.Id, roundTrip.Id)
	assert.Equal(t, msg.Namespace, roundTrip.Namespace)
}

// Benchmarks: fastjson vs protojson (no database, pure marshal/unmarshal)

func BenchmarkMarshal_Protojson(b *testing.B) {
	msg := makeTestIndicator()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := protojson.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_FastJSON(b *testing.B) {
	msg := makeTestIndicator()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := msg.MarshalFastJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_Protojson(b *testing.B) {
	msg := makeTestIndicator()
	data, _ := protojson.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		out := &ProcessIndicatorJsonb{}
		if err := protojson.Unmarshal(data, out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_FastJSON(b *testing.B) {
	msg := makeTestIndicator()
	data, _ := msg.MarshalFastJSON()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		out := &ProcessIndicatorJsonb{}
		if err := out.UnmarshalFastJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundTrip_Protojson(b *testing.B) {
	msg := makeTestIndicator()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		data, err := protojson.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
		out := &ProcessIndicatorJsonb{}
		if err := protojson.Unmarshal(data, out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundTrip_FastJSON(b *testing.B) {
	msg := makeTestIndicator()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		data, err := msg.MarshalFastJSON()
		if err != nil {
			b.Fatal(err)
		}
		out := &ProcessIndicatorJsonb{}
		if err := out.UnmarshalFastJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}
