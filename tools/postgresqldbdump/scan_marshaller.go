package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/proto"
)

const (
	defaultMaxLineSize = 1024 * 4096
	maxEntries         = 1000
)

var (
	foundUnhandledTables = set.NewStringSet()
	fatalErrorsFound     bool
)

// DataRow represents a parsed entry with fields and serialized content
type DataRow struct {
	// Fields contains the all non serialized fields of a DB row
	Fields json.RawMessage `json:"fields,inline,omitempty"`
	// Serialized contains the deserialized proto message
	Serialized json.RawMessage `json:"serialized,inline,omitempty"`
}

// ScanMarshaller encapsulates the scanning and marshal logic
type ScanMarshaller struct {
	scanner *bufio.Scanner

	rows          []*DataRow
	currentTable  string
	hasSerialized bool
	index         int
	header        string
	outputDir     string
	metadataFile  *os.File
}

// NewScanMashaller creates and initializes a new ScanMarshaller
func NewScanMashaller(r io.Reader, outputDir string) (*ScanMarshaller, error) {
	// Set buffer size for the scanner
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, defaultMaxLineSize)
	metadataPath := filepath.Join(outputDir, "metadata.txt")
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metadata file")
	}

	return &ScanMarshaller{
		scanner:      scanner,
		outputDir:    outputDir,
		metadataFile: metadataFile,
	}, nil
}

// ScanMarshal processes the input data
func (ds *ScanMarshaller) ScanMarshal() error {
	const (
		dataPrefix = "-- Data for Name: "
		copyPrefix = "COPY public."
		copySuffix = ", serialized) FROM stdin;"
		lineBreak  = "\\."
	)

	for ds.scanner.Scan() {
		line := strings.TrimSpace(ds.scanner.Text())
		if line == "" || line == "--" {
			continue
		}

		// Catch table name
		if trimmed, hasPrefix := stringutils.MaybeTrimPrefix(line, dataPrefix); hasPrefix {
			ds.currentTable, _ = stringutils.Split2(trimmed, ";")
			continue
		}

		// Handle Data header
		if strings.HasPrefix(line, copyPrefix) {
			ds.header = line + "\n"
			ds.hasSerialized = strings.HasSuffix(line, copySuffix)
			continue
		}

		// Handle end of table data
		if line == lineBreak {
			if err := ds.flushData(); err != nil {
				return err
			}
			ds.endOfTable()
			continue
		}

		// Process metadata or table rows
		if ds.currentTable == "" || ds.header == "" {
			if _, err := io.WriteString(ds.metadataFile, ds.scanner.Text()+"\n"); err != nil {
				return errors.Wrap(err, "failed to write metadata")
			}
			continue
		}

		// Process a data row
		if err := ds.processRow(line); err != nil {
			fmt.Printf("Warning: failed to process row for table %s: %v line:%s\n", ds.currentTable, err, line)
		}
	}
	return ds.scanner.Err()
}

// processRow parses a line into a DataRow and appends it to rows
func (ds *ScanMarshaller) processRow(line string) error {
	row, err := createDataRow(ds.header, line, ds.currentTable, ds.hasSerialized)
	if err != nil {
		return err
	}
	ds.rows = append(ds.rows, row)

	if len(ds.rows) >= maxEntries {
		if err := ds.flushData(); err != nil {
			return err
		}
		ds.rows = ds.rows[:0]
	}
	return nil
}

// flushData writes the current table data to a JSON file
func (ds *ScanMarshaller) flushData() error {
	if len(ds.rows) == 0 {
		return nil
	}
	suffix := ""
	if ds.index > 0 {
		suffix = fmt.Sprintf("_%d", ds.index)
	}
	fileName := filepath.Join(ds.outputDir, fmt.Sprintf("%s%s.json", ds.currentTable, suffix))
	ds.index++
	return marshalToJson(ds.rows, fileName)
}

// endOfTable resets the scanner state for a new table
func (ds *ScanMarshaller) endOfTable() {
	ds.rows = ds.rows[:0]
	ds.currentTable = ""
	ds.hasSerialized = false
	ds.index = 0
	ds.header = ""
}

// Close cleans up resources used by ScanMarshaller
func (ds *ScanMarshaller) Close() error {
	return ds.metadataFile.Close()
}

// createDataRow converts a row entry into a DataRow and handles serialization.
func createDataRow(headerLine, valueLine, tableName string, hasSerialized bool) (*DataRow, error) {
	je := &DataRow{}
	var rest, serialized string
	if hasSerialized {
		// We have to depend on the fact that serialized field is always the last field
		rest, serialized = stringutils.Split2Last(valueLine, "\\\\x")
	} else {
		rest = valueLine
	}
	fields, err := genFields(headerLine, rest)
	if err != nil {
		return nil, err
	}
	// Marshal Fields to JSON
	je.Fields, err = json.Marshal(fields)
	if err != nil {
		return nil, errors.Errorf("failed to marshal Fields: %v", err)
	}

	possibleObjectID, _ := stringutils.Split2(rest, "\t")
	if possibleObjectID == "" {
		possibleObjectID = "NoID"
	}
	if serialized != "" {
		decoded, err := hex.DecodeString(serialized)
		if err != nil {
			log.Infof("Failed to decode (text=%q): %v\n", serialized, err)
			return je, err
		}

		pbInterface, ok := getProtoMessage(tableName)
		if !ok {
			handleUnknownTable(tableName)
			return je, err
		}
		pb, err := unmarshalProto(decoded, pbInterface, tableName, possibleObjectID)
		if err != nil {
			return nil, err
		}
		jsonStr, err := jsonutil.MarshalToString(*pb)
		if err != nil {
			return nil, err
		}
		je.Serialized = []byte(jsonStr)
	}
	return je, err
}

// marshalToJson marshals a list of DataRow structs to a JSON file.
func marshalToJson(entries []*DataRow, outputPath string) error {
	if len(entries) == 0 {
		return nil
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return errors.Errorf("failed to create file: %v", err)
	}
	defer utils.IgnoreError(file.Close)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Set indentation for pretty printing

	return encoder.Encode(entries)
}

// unmarshalProto unmarshal a protobuf message from bytes.
func unmarshalProto(data []byte, pbInterface protocompat.Message, tableName, objectID string) (*protocompat.Message, error) {
	pbType := reflect.TypeOf(pbInterface)
	value := reflect.New(pbType.Elem()).Interface()
	pb, _ := value.(protocompat.Message)
	if err := proto.Unmarshal(data, pb); err != nil {
		log.Infof("Cannot unmarshal table: %s, ID: %s, error: %v\n", tableName, objectID, err)
		return nil, err
	}
	return &pb, nil
}

// genFields parses a COPY command into a map of column names and values.
func genFields(headerLine, valueLine string) (map[string]string, error) {
	start := strings.Index(headerLine, "(")
	end := strings.Index(headerLine, ")")
	if start == -1 || end == -1 || start >= end {
		return nil, errors.New("invalid COPY command format")
	}

	columnPart := headerLine[start+1 : end]
	columns := strings.Split(columnPart, ", ")
	values := strings.Split(valueLine, "\t")
	if len(values) > len(columns) {
		return nil, errors.New("number of values are more than number of columns")
	}

	result := make(map[string]string)
	for i, column := range columns {
		if i < len(values) {
			result[column] = values[i]
		} else {
			result[column] = ""
		}
	}
	delete(result, "serialized")
	return result, nil
}

// handleUnknownTable logs and manages missing table schemas.
func handleUnknownTable(tableName string) {
	if added := foundUnhandledTables.Add(tableName); added {
		log.Errorf("Table %s is missing from the protobuf map\n", tableName)
	}
	fatalErrorsFound = fatalErrorsFound || !knownUnhandledBuckets.Contains(tableName)
}
