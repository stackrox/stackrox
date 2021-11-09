package datastore

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	_ "github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
)

func TestT(t *testing.T) {
	//source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	//db, err := sql.Open("postgres", source)
	//if err != nil {
	//	panic(err)
	//}
	//defer db.Close()
	//
	//processStore := store.New(db)
	//fmt.Println(processStore)
	//
	//processIndex := index.NewIndexer(db)
	//fmt.Println(processIndex)
	//
	//var processes []*storage.ProcessIndicator
	//for i := 0; i < 100000; i++ {
	//	process := fixtures.GetProcessIndicator()
	//	process.Id = uuid.NewV4().String()
	//	processes = append(processes, process)
	//}
	//if err := processStore.UpsertMany(processes); err != nil {
	//	panic(err)
	//}
}

const data = `{
  "id": "47954890-792a-5edd-b639-a7e6931d1d89",
  "podId": "172ed8e39d6c7a61080e9f50d3488024d891d0943c4abcaa",
  "podUid": "09c3d035-6d37-59b7-a871-a01ef094d31f",
  "signal": {
    "id": "",
    "gid": 0,
    "pid": 0,
    "uid": 0,
    "args": "abc def ghi jkl lmn op qrs tuv",
    "name": "wc",
    "time": "2021-11-03T22:47:23.863651928Z",
    "lineage": [],
    "scraped": false,
    "containerId": "6e9fb446b58b",
    "lineageInfo": [
      {
        "parentUid": 0,
        "parentExecFilePath": "java"
      },
      {
        "parentUid": 0,
        "parentExecFilePath": "bash"
      }
    ],
    "execFilePath": "/bin/wc"
  },
  "imageId": "sha256:d72bedfeb461346771e17d20fabf882816c593e385ae74746d1881a5faee06fb",
  "clusterId": "307f1572-2419-42fb-9e34-e8228b2a15b6",
  "namespace": "5acfeb2a79b26c86",
  "deploymentId": "feacdf65-77d6-4040-8336-c4748b34de34",
  "containerName": "303ba3c7ecf176e68fffad6c18fad7aada0767c654ea1af9",
  "containerStartTime": null
}`

func BenchmarkProcessMarshal(b *testing.B) {
	marshaler := &jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true}

	var indicator storage.ProcessIndicator
	if err := jsonpb.UnmarshalString(data, &indicator); err != nil {
		panic(err)
	}

	b.ResetTimer()
	var s string
	for i := 0; i < b.N; i++ {
		s, _ = marshaler.MarshalToString(&indicator)
	}
	fmt.Println(s)
}
