package jsoncache

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/u6du/go-rfc1924/base85"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type JSONCache struct {
	dir string
}

// New creates a new JSONCache.
func New(dir string) *JSONCache {
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		log.Fatalf("cannot create cache directory: %v", err)
	}
	return &JSONCache{dir: dir}
}

func (c *JSONCache) pathFor(key string) string {
	// slug + digest to keep filenames short & safe
	slug := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(strings.ToLower(key), "")
	sum := sha1.Sum([]byte(key))
	digest := base85.EncodeToString(sum[:])
	return filepath.Join(c.dir, fmt.Sprintf("%s-%s.json", slug, digest))
}

func (c *JSONCache) Get(key string, v any) (bool, error) {
	path := c.pathFor(key)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, json.Unmarshal(b, v)
}

func (c *JSONCache) Set(key string, v any) error {
	path := c.pathFor(key)
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
