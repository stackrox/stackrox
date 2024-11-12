package main

import (
	"archive/zip"
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/proto"
)

const (
	defaultMaxLineSize = 1024 * 4096
	metadataName       = "metadata.txt"
	postgresDumpPath   = "postgres.dump"
)

var (
	foundUnhandledTables = set.NewStringSet()
	fatalErrorsFound     bool
	log                  = logging.LoggerForModule()
)

type Parameters struct {
	// InputFile points to the Central DB dump file or the Postgres DB dump.
	InputFile string
	// OutputFile is the path to dump the decoded files.
	OutputDir string
	// DBDump set to true if the InputFile is a Postgres DB dump.
	DBDump bool
}

// DataRow represents a parsed entry with fields and serialized content
type DataRow struct {
	// Fields contains the all non serialized fields of a DB row
	Fields json.RawMessage `json:"fields,inline,omitempty"`
	// Serialized contains the deserialized proto message
	Serialized json.RawMessage `json:"serialized,inline,omitempty"`
}

func main() {
	input := Parameters{}
	cmd := &cobra.Command{
		Use:   "postgresdbdump <backup file in zip | postgres db dump file>",
		Args:  cobra.ExactArgs(1),
		Short: "Dump postgres DB",
		Long:  "Dump postgres DB into JSON files",
		RunE: func(cmd *cobra.Command, args []string) error {
			input.InputFile = args[0]
			if input.DBDump {
				return processDBDumpFile(input.InputFile, input.OutputDir)
			}
			return processCentralBackup(input.InputFile, input.OutputDir)
		},
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVarP(&input.OutputDir, "output-dir", "o", "", "Directory for output files (must exist)")
	cmd.PersistentFlags().BoolVarP(&input.DBDump, "db-dump", "d", false, "Process the file as db dump")
	utils.Must(cmd.MarkPersistentFlagRequired("output-dir"))

	if err := cmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
	log.Infof("Generated %s", input.OutputDir)
}

// processCentralBackup reads and processes each entry in the ZIP archive
func processCentralBackup(zipPath, outputDir string) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Errorf("failed to open zip file: %v", err)
	}
	defer utils.IgnoreError(zipReader.Close)
	for _, file := range zipReader.File {
		if file.Name == postgresDumpPath {
			if err := extractDBDump(file, outputDir); err != nil {
				log.Errorf("failed to deserialize postgres dump: %v", err)
			}
			continue
		}
		if err := extractZipEntry(file, outputDir); err != nil {
			log.Errorf("Error processing file %s: %v\n", file.Name, err)
		}
	}
	return nil
}

func processDBDumpFile(path, outputDir string) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Errorf("failed to open file %s", path)
	}
	defer utils.IgnoreError(file.Close)
	return pgRestore(file, outputDir)
}

// extractZipEntry saves the generated JSON files to a specific output directory.
func extractDBDump(file *zip.File, outputDir string) error {
	dbDump, err := file.Open()
	if err != nil {
		return errors.Errorf("failed to open %s in zip", postgresDumpPath)
	}
	defer utils.IgnoreError(dbDump.Close)
	dumpPath := filepath.Join(outputDir, postgresDumpPath)
	if err := os.MkdirAll(dumpPath, 0755); err != nil {
		return errors.Errorf("failed to create directories for %s: %v", dumpPath, err)
	}
	return pgRestore(dbDump, dumpPath)
}

// extractZipEntry saves each entry in the ZIP archive to the specified output directory.
func extractZipEntry(file *zip.File, outputDir string) error {
	rc, err := file.Open()
	if err != nil {
		return errors.Errorf("failed to open file %s in zip: %v", file.Name, err)
	}
	defer utils.IgnoreError(rc.Close)

	outputPath := filepath.Join(outputDir, file.Name)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", outputPath, err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return errors.Errorf("failed to create file %s: %v", outputPath, err)
	}
	defer utils.IgnoreError(outFile.Close)

	_, err = io.Copy(outFile, rc)
	return err
}

func pgRestore(file io.Reader, outputDir string) error {
	pr, pw := io.Pipe()
	go func() {
		// close the writer, so the reader knows there's no more data
		defer utils.IgnoreError(pw.Close)

		// write json data to the PipeReader through the PipeWriter
		if _, err := io.Copy(pw, file); err != nil {
			log.Fatalf("failed to read postgres dump", err)
		}
	}()

	cmd := exec.Command("pg_restore", "-f", "-")
	cmd.Stdin = pr

	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err := scan(output, outputDir)
		if err != nil {
			log.Error(err)
		}
	}()
	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	return nil
}

// scan scans the restored PostgresSQL dump and writes data to JSON files.
func scan(r io.Reader, outputDir string) error {
	metadata, err := os.Create(filepath.Join(outputDir, metadataName))
	if err != nil {
		return errors.Errorf("failed to create metadata file: %v", err)
	}
	defer utils.IgnoreError(metadata.Close)

	tableData := make(map[string][]*DataRow)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, defaultMaxLineSize)
	var currentTable, header string
	var hasSerialized bool

	for scanner.Scan() {
		origText := scanner.Text()
		line := strings.TrimSpace(origText)
		if line == "" || line == "--" {
			continue
		}

		// Identify table name in COPY statement
		if trimmed, hasPrefix := stringutils.MaybeTrimPrefix(line, "-- Data for Name: "); hasPrefix {
			currentTable, _ = stringutils.Split2(trimmed, ";")
			continue
		}
		if strings.HasPrefix(line, "COPY public.") {
			header = line + "\n"
			hasSerialized = strings.HasSuffix(line, ", serialized) FROM stdin;")
			continue
		}
		if line == "\\." {
			err := marshalToJson(tableData[currentTable], filepath.Join(outputDir, fmt.Sprintf("%s.json", currentTable)))
			if err != nil {
				return err
			}
			currentTable = ""
			hasSerialized = false
			continue
		}
		if currentTable == "" {
			_, err := io.WriteString(metadata, origText+"\n")
			if err != nil {
				return err
			}
			continue
		}

		// Process entries with known table schema
		if _, exists := getProtoInterface(currentTable); exists {
			row, err := genDataRow(header, line, currentTable, hasSerialized)
			if err != nil {
				fmt.Printf("Warning: failed to parse row for table %s: %v\n", currentTable, err)
				continue
			}
			tableData[currentTable] = append(tableData[currentTable], row)
		} else {
			handleUnknownTable(currentTable)
		}
	}
	return scanner.Err()
}

// genDataRow converts a row entry into a DataRow and handles serialization.
func genDataRow(headerLine, valueLine, tableName string, hasSerialized bool) (*DataRow, error) {
	je := &DataRow{}
	var rest, serialized string
	if hasSerialized {
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

		pbInterface, ok := getProtoInterface(tableName)
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

// handleUnknownTable logs and manages missing table schemas.
func handleUnknownTable(tableName string) {
	if added := foundUnhandledTables.Add(tableName); added {
		log.Errorf("Table %s is missing from the protobuf map\n", tableName)
	}
	fatalErrorsFound = fatalErrorsFound || !knownUnhandledBuckets.Contains(tableName)
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
