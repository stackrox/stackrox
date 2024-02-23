package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

var protobufType *string = flag.String("type", "", "name of protobuf, e.g., storage.Alert")

func main() {
	flag.Parse()

	if *protobufType == "" {
		log.Fatal("must provide --type")
	}
	message := reflect.New(typeRegistry[*protobufType]).Elem()
	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		//log.Println(reader.Text())
		s, err := strconv.Unquote(fmt.Sprintf("\"%s\"", reader.Text()))
		if err != nil {
			panic(err)
		}

		s = "0A" + strings.TrimSpace(s)

		b, err := hex.DecodeString(s)
		if err != nil {
			panic(err)
		}

		v := []reflect.Value{reflect.ValueOf(b)}
		pe := message.Addr().MethodByName("Unmarshal").Call(v)[0]
		if !pe.IsNil() {
			err2 := pe.Interface().(error)
			if err2 != nil {
				panic(err2)
			}
		}

		//log.Println(message.Addr().MethodByName("String").Call([]reflect.Value{}))
		pjson, err := json.MarshalIndent(message.Interface(), "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(pjson))
	}

	if err := reader.Err(); err != nil {
		log.Fatalf("error while reading from stdin: %v", err)
	}
}
