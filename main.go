package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
)

func main() {
	req, err := os.ReadFile("request.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(req))

	buffer := bytes.NewBuffer(req)
	notifier := &storage.Notifier{}

	err = jsonpb.Unmarshal(buffer, notifier)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("success")
}
