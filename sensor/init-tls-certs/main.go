package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	destinationDir  string
	legacySourceDir string
	newSourceDir    string
)

func init() {
	flag.StringVar(&legacySourceDir, "legacy", "/tmp/certs-legacy", "Source directory for legacy certs")
	flag.StringVar(&newSourceDir, "new", "/tmp/certs-new", "Source directory for new certs")
	flag.StringVar(&destinationDir, "destination", "/tmp/certs", "Destination directory for certs")
}

func main() {
	flag.Parse()

	fmt.Println("New certs directory:", newSourceDir)
	fmt.Println("Legacy certs directory:", legacySourceDir)
	fmt.Println("Destination directory:", destinationDir)

	realDest, err := sanityCheckDestination()
	if err != nil {
		log.Fatalf("Cannot check destination directory %q: %s", destinationDir, err)
	}
	log.Printf("Destination directory %q looks sane.", destinationDir)

	timeout := 5 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	files, err := waitForSource(ctx)
	if err != nil {
		log.Fatalf("Failed to load certificates: %v", err)
	}

	if err = copyFiles(files, realDest); err != nil {
		log.Fatalf("Cannot copy files: %v", err)
	}
}

func copyFiles(files []string, destDir string) error {
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		destPath := path.Join(destDir, path.Base(file))
		perm := os.FileMode(0600) // 0600 is required by Postgres (used by scanner-db)
		if err = os.WriteFile(destPath, content, perm); err != nil {
			return err
		}
		log.Printf("Copied %q to %q", file, destPath)
	}
	return nil
}

func waitForSource(ctx context.Context) ([]string, error) {
	log.Printf("Looking for files in the source directory %q.", legacySourceDir)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context expired while waiting for certificates: %w", ctx.Err())
		default:
			// Check new certificates first
			files, err := findFiles(newSourceDir)
			if err != nil {
				log.Printf("Error checking certificates in %q: %s", newSourceDir, err)
			} else {
				log.Printf("Using %d new certificate files from %q.", len(files), newSourceDir)
				return files, nil
			}

			// Fall back to legacy certificates
			files, err = findFiles(legacySourceDir)
			if err != nil {
				log.Printf("Error checking legacy certificates in %q: %s", legacySourceDir, err)
			} else {
				log.Printf("Using %d legacy certificates from %q.", len(files), legacySourceDir)
				return files, nil
			}

			log.Printf("No certificates found. Retrying...")
			time.Sleep(5 * time.Second)
		}
	}
}

func findFiles(sourceDir string) ([]string, error) {
	realSource, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return nil, err
	}

	log.Printf("Walking %q.", realSource)
	var files []string
	err = filepath.WalkDir(realSource, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Printf("Error accessing path %q: %s", path, walkErr)
			return nil
		}

		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			if d.IsDir() {
				log.Printf("Ignoring hidden dir %q", path)
				return filepath.SkipDir
			}

			log.Printf("Ignoring hidden file %q", path)
			return nil
		}

		realFile, err := filepath.EvalSymlinks(path)
		if err != nil {
			log.Printf("Ignoring file %q: %s", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		log.Printf("Found file %q (%q)", path, realFile)
		files = append(files, realFile)
		return nil
	})

	if err != nil {
		return nil, err
	}

	requiredFiles := 3
	if len(files) < requiredFiles {
		return nil, fmt.Errorf("expecting at least %d files at %q", requiredFiles, sourceDir)
	}

	return files, nil
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
		return "", fmt.Errorf("%q is not a directory", realDest)
	}
	return realDest, nil
}
