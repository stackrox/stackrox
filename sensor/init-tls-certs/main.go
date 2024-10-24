package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	destinationDir  = "/tmp/certs"
	legacySourceDir = "/tmp/certs-legacy"
)

func main() {
	log.Print("Started.")
	realDest, err := sanityCheckDestination()
	if err != nil {
		log.Fatalf("Cannot check destination directory %q: %s", destinationDir, err)
	}
	log.Printf("Destination directory %q looks sane.", destinationDir)
	files, err := waitForSource()
	if err != nil {
		log.Fatalf("Cannot find source files in %q: %s", legacySourceDir, err)
	}
	if err = copyFiles(files, realDest); err != nil {
		log.Fatalf("Cannot copy files: %s", err)
	}
}

func copyFiles(files []string, destDir string) error {
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		destPath := path.Join(destDir, path.Base(file))
		if err = os.WriteFile(destPath, content, 0666); err != nil {
			return err
		}
		log.Printf("Copied %q to %q", file, destPath)
	}
	return nil
}

func waitForSource() ([]string, error) {
	log.Printf("Looking for files in the source directory %q.", legacySourceDir)
	for {
		realSource, err := filepath.EvalSymlinks(legacySourceDir)
		if err != nil {
			return nil, err
		}
		log.Printf("Walking %q.", realSource)
		var files []string
		err = filepath.WalkDir(realSource, func(path string, d fs.DirEntry, err error) error {
			if strings.HasPrefix(path, ".") {
				log.Printf("Ignoring hidden file %q", path)
				return nil
			}
			realFile, err := filepath.EvalSymlinks(path)
			if err != nil {
				log.Printf("Ignoring file %q: %s", path, err)
				return nil
			}
			st, err := os.Stat(realFile)
			if err != nil {
				log.Printf("Ignoring file %q: %s", realFile, err)
				return nil
			}
			if st.IsDir() {
				return nil
			}
			log.Printf("Found file %q (%q)", path, realFile)
			files = append(files, realFile)
			return nil
		})
		if err != nil {
			return nil, err
		}
		if len(files) >= 3 {
			return files, nil
		}
		log.Printf("Did not find (enough) files, sleeping.")
		time.Sleep(time.Second)
	}
}

func sanityCheckDestination() (string, error) {
	realDest, err := filepath.EvalSymlinks(destinationDir)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(realDest)
	if err != nil {
		return "", fmt.Errorf("stat failed: %w", err)
	}
	if !st.IsDir() {
		return "", errors.New("%q is not a directory")
	}
	return realDest, nil
}
