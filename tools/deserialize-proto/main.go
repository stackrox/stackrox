package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	flag "github.com/spf13/pflag"
	_ "github.com/stackrox/rox/generated/storage"
)

var protobufType *string = flag.String("type", "", "name of protobuf, e.g., storage.Alert")

func main() {
	flag.Parse()

	if *protobufType == "" {
		log.Fatal("must provide --type")
	}

	mt := proto.MessageType(*protobufType)
	if mt == nil {
		log.Fatalf("type %s could not be resolved to a protobuf message type", *protobufType)
	}
	msg := reflect.New(mt.Elem()).Interface().(proto.Message)

	var text string

	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		text = reader.Text()
		if len(text) == 0 {
			break
		}

		// It's not clear why we need to both unquote *and* prepend 0A but it works ¯\_(ツ)_/¯
		s, err := strconv.Unquote(fmt.Sprintf("\"%s\"", text))
		if err != nil {
			log.Fatalf("error while unquote the serialized text, text = %s, err = %v", text, err)
		}

		s = "0A" + strings.TrimSpace(s)

		b, err := hex.DecodeString(s)
		if err != nil {
			log.Fatalf("error while decoding the hex value of the serialized text, text = %s, err = %v",
				text, err)
		}

		if err := proto.Unmarshal(b, msg); err != nil {
			log.Fatalf("error unmarshaling proto, text = %s, err = %v", text, err)
		}

		m := jsonpb.Marshaler{Indent: "  "}
		json, err := m.MarshalToString(msg)
		if err != nil {
			log.Fatalf("error while prettifying, message = %+v, err = %v", msg, err)
		}
		fmt.Println(json)
	}

	if err := reader.Err(); err != nil {
		log.Fatalf("error while reading from stdin, err = %v", err)
	}
}
