package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"k8s.io/utils/env"

	// This will register all proto types in the package,
	// making it possible to retrieve the message type by name.
	_ "github.com/stackrox/rox/generated/storage"
)

var (
	protobufType = flag.String("type", "", "name of protobuf, e.g., storage.Alert")
	id           = flag.String("id", "", "id of the object to query")
	whereClause  = flag.String("where", "", "additional where clause for the objects to query")
	fromStdin    = flag.Bool("stdin", false, "reads serialized protos from stdin")
	debug        = flag.Bool("debug", false, "enable debug logging")

	errUnqoting  = errors.New("failed to unquote the serialized text")
	errDecoding  = errors.New("failed decoding the hex value of the text")
	errUnmarshal = errors.New("failed unmarshalling the proto")
)

func main() {
	flag.Parse()

	if *protobufType == "" {
		log.Fatal("must provide --type")
	}

	mt := protoutils.MessageType(*protobufType)
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
	dbName := env.GetString("POSTGRES_DATABASE", "central")
	connectionString := conn.GetConnectionStringWithDatabaseName(&testing.T{}, dbName)

	if *debug {
		fmt.Println("Connecting to database: ", connectionString)
	}

	db, err := postgres.Connect(context.TODO(), connectionString)
	utils.Should(err)

	tableName := pgutils.NamingStrategy.TableName(stringutils.GetAfter(*protobufType, "."))

	query := fmt.Sprintf("SELECT serialized FROM %s", tableName)
	if *id != "" && *whereClause == "" {
		query = fmt.Sprintf("%s WHERE id = '%s'", query, *id)
	} else if *whereClause != "" {
		query = fmt.Sprintf("%s WHERE %s", query, *whereClause)
	} else if *whereClause != "" && *id != "" {
		log.Fatalf("cannot provide both id and where clause")
	}

	if *debug {
		fmt.Printf("Run query: %s", query)
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

		m := protojson.MarshalOptions{Indent: "  ", EmitDefaultValues: true}
		json, err := m.Marshal(msg)
		if err != nil {
			utils.Should(err)
		}
		fmt.Println(string(json))
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

		m := protojson.MarshalOptions{EmitUnpopulated: true, EmitDefaultValues: true}
		j, err := m.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed marshalling the proto to JSON (msg=%+v): %w", msg, err)
		}
		dst := bytes.Buffer{}
		err = json.Indent(&dst, j, "", "  ")
		if err != nil {
			return fmt.Errorf("failed indenting the JSON %q: %w", j, err)
		}
		if _, err := fmt.Fprintln(out, dst.String()); err != nil {
			return fmt.Errorf("failed writing proto JSON to output: %w", err)
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("reading from input: %w", err)
	}
	return nil
}
