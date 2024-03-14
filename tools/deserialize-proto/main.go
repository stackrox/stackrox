package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	flag "github.com/spf13/pflag"
)

var protobufType *string = flag.String("type", "", "name of protobuf, e.g., storage.Alert")

func unmarshalProto[T proto.Unmarshaler](t T, b []byte) error {
	return t.Unmarshal(b)
}

func main() {
	flag.Parse()

	if *protobufType == "" {
		log.Fatal("must provide --type")
	}

	messageType, ok := typeRegistry[*protobufType]
	if !ok {
		log.Fatalf("%s is an invalid storage type", *protobufType)
	}

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

		message := messageType
		if err := unmarshalProto(message, b); err != nil {
			log.Fatalf("error unmarshaling proto, text = %s, err = %v", text, err)
		}

		pjson, err := json.MarshalIndent(message, "", "  ")
		if err != nil {
			log.Fatalf("error while prettifying, message = %s, err = %v", message, err)
		}
		fmt.Println(string(pjson))
	}

	if err := reader.Err(); err != nil {
		log.Fatalf("error while reading from stdin, err = %v", err)
	}
}
