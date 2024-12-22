package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	postgresDumpPath = "postgres.dump"
)

var log = logging.LoggerForModule()

type Parameters struct {
	// InputFile points to the Central DB dump file or the Postgres DB dump.
	InputFile string
	// OutputFile is the path to dump the decoded files.
	OutputDir string
	// DBDump set to true if the InputFile is a Postgres DB dump.
	DBDump bool
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
	tmpFile := filepath.Join(outputDir, "dbDump")
	cmd := exec.Command("pg_restore", "-f", tmpFile)
	cmd.Stdin = pr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile)
	}()
	rc, err := os.Open(tmpFile)
	if err != nil {
		log.Fatal(err)
	}
	defer utils.IgnoreError(rc.Close)
	worker, err := NewScanMashaller(rc, outputDir)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(worker.Close)
	return worker.ScanMarshal()
}
