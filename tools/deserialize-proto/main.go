package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	flag "github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"

	// This will register all proto types in the package,
	// making it possible to retrieve the message type by name.
	_ "github.com/stackrox/rox/generated/storage"
)

var (
	protobufType = flag.String("type", "", "name of protobuf, e.g., storage.Alert")
	id           = flag.String("id", "", "id of the object to query")
	whereClause  = flag.String("where-clause", "", "additional where clause for the objects to query")
	fromStdin    = flag.Bool("from-stdin", false, "reads serialized protos from stdin")

	errUnqoting  = errors.New("failed to unquote the serialized text")
	errDecoding  = errors.New("failed decoding the hex value of the text")
	errUnmarshal = errors.New("failed unmarshalling the proto")
)

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

	// reads the serialized protos directly from stdin
	if *fromStdin {
		utils.Should(printProtoMessagesFromStdin(os.Stdin, os.Stdout, msg))
		return
	}

	// reads directly from the database.
	readFromDatabase(msg)
}

// Detect database directly from provided proto. Add id or optional where clauses to the SQL query.
func readFromDatabase(msg proto.Message) {

	db, err := postgres.Connect(context.TODO(), "postgres://postgres:password@localhost:5432/central")
	utils.Should(err)

	tableName, _ := strings.CutPrefix(stringutils.ToSnakeCase(*protobufType), "storage._")

	query := fmt.Sprintf("SELECT serialized FROM %s", tableName)
	if *id != "" && *whereClause == "" {
		query = fmt.Sprintf("%s WHERE id = '%s'", query, *id)
	} else {
		query = fmt.Sprintf("%s WHERE id = '%s' AND %s", query, *id, *whereClause)
	}

	rows, err := db.Query(context.TODO(), query)
	if err != nil {
		fmt.Println("SQL QUERY: ", query)
		utils.Should(err)
	}

	for rows.Next() {
		value, err := rows.Values()
		utils.Should(err)

		if err := proto.Unmarshal(value[0].([]byte), msg); err != nil {
			utils.Should(err)
		}

		m := jsonpb.Marshaler{Indent: "  "}
		json, err := m.MarshalToString(msg)
		if err != nil {
			utils.Should(err)
		}
		fmt.Println(json)
	}
}

func printProtoMessagesFromStdin(in io.Reader, out io.Writer, msg proto.Message) error {
	reader := bufio.NewScanner(in)

	for reader.Scan() {
		text := reader.Text()
		if len(text) == 0 {
			break
		}

		// It's not clear why we need to both unquote *and* prepend 0A but it works ¯\_(ツ)_/¯
		s, err := strconv.Unquote(fmt.Sprintf("\"%s\"", text))
		if err != nil {
			return fmt.Errorf("%w (text=%q): %w", errUnqoting, text, err)
		}

		s = "0A" + strings.TrimSpace(s)

		b, err := hex.DecodeString(s)
		if err != nil {
			return fmt.Errorf("%w (text=%q): %w", errDecoding, text, err)
		}

		if err := proto.Unmarshal(b, msg); err != nil {
			return fmt.Errorf("%w: %w", errUnmarshal, err)
		}

		m := jsonpb.Marshaler{Indent: "  "}
		json, err := m.MarshalToString(msg)
		if err != nil {
			return fmt.Errorf("failed marshalling the proto to JSON (msg=%+v): %w", msg, err)
		}

		if _, err := fmt.Fprintln(out, json); err != nil {
			return fmt.Errorf("failed writing proto JSON to output: %w", err)
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("reading from input: %w", err)
	}
	return nil
}
