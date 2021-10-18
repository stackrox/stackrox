package datastore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"
	"github.com/stackrox/rox/generated/storage"
)

func getAlert() *storage.Alert {
	data, err := ioutil.ReadFile("/Users/connorgorman/repos/src/github.com/stackrox/rox/worst-case-alert.json")
	if err != nil {
		panic(err)
	}
	var alert storage.Alert
	if err := jsonpb.Unmarshal(bytes.NewBuffer(data), &alert); err != nil {
		panic(err)
	}
	return &alert
}

func TestT1(t *testing.T) {
	alert := getAlert()
	str, err := json.MarshalIndent(alert, "", "  ")
	fmt.Println(string(str), err)
}

func BenchmarkMarshalJSONPB(b *testing.B) {
	alert := getAlert()
	b.ResetTimer()

	marshaler := &jsonpb.Marshaler{}

	for i := 0; i < b.N; i++ {
		marshaler.MarshalToString(alert)
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	alert := getAlert()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jsoniter.MarshalToString(alert)
	}
}


func BenchmarkUnmarshalJSONPB(b *testing.B) {
	alert := getAlert()

	marshaler := &jsonpb.Marshaler{}
	str, _ := marshaler.MarshalToString(alert)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jsonpb.Unmarshal(bytes.NewBufferString(str), alert)
	}
}
//
//func BenchmarkUnmarshalJSON(b *testing.B) {
//	alert := getAlert()
//
//	str, _ := json.MarshalToString(alert)
//	byt := []byte(str)
//	b.ResetTimer()
//
//	for i := 0; i < b.N; i++ {
//		json.Unmarshal(byt, alert)
//	}
//}
