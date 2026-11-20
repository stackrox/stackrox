package key

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeDeterministic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package util"), 0o644)

	k1, err := Compute(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := Compute(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if k1 != k2 {
		t.Fatalf("same tree should produce same key: %s != %s", k1, k2)
	}
}

func TestComputeChangesOnFileChange(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)

	k1, _ := Compute(dir, nil)

	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main // changed"), 0o644)
	k2, _ := Compute(dir, nil)

	if k1 == k2 {
		t.Fatal("key should change when file content changes")
	}
}

func TestComputeIgnoresHidden(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("should be ignored"), 0o644)

	k1, _ := Compute(dir, nil)

	// Change the hidden file
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("changed"), 0o644)
	k2, _ := Compute(dir, nil)

	if k1 != k2 {
		t.Fatal("hidden files should not affect the key")
	}
}

func TestComputeWithExclusion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "gen.go"), []byte("generated"), 0o644)

	k1, _ := Compute(dir, []string{"gen.go"})

	// Change the excluded file
	os.WriteFile(filepath.Join(dir, "gen.go"), []byte("changed"), 0o644)
	k2, _ := Compute(dir, []string{"gen.go"})

	if k1 != k2 {
		t.Fatal("excluded files should not affect the key")
	}
}

func TestCommandIncludesCommand(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)

	k1, _ := Command(dir, "go build", nil)
	k2, _ := Command(dir, "go test", nil)

	if k1 == k2 {
		t.Fatal("different commands should produce different keys")
	}
}
