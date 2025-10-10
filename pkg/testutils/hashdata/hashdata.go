package hashdata

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed image.json
var testImageJsonBytes []byte

func GetImage() (*storage.Image, error) {
	var image storage.Image
	if err := protojson.Unmarshal(testImageJsonBytes, &image); err != nil {
		return &image, fmt.Errorf("failed to unmarshal image data: %w", err)
	}

	return &image, nil
}

func WriteLinesToFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to write golden file at path: %s, err: %w", filePath, err)
	}
	defer utils.IgnoreError(file.Close)

	writer := bufio.NewWriter(file)

	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("error writing to golden file: %s, err: %w", filePath, err)
		}
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush content to file: %s, err: %w", filePath, err)
	}

	return nil
}

func ReadLinesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open golden file at path: %s, err: %w", filePath, err)
	}
	defer utils.IgnoreError(file.Close)

	var ids []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			ids = append(ids, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading golden file: %s, err: %w", filePath, err)
	}

	return ids, nil
}
