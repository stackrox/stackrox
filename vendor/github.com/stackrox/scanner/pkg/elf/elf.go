package elf

import (
	"debug/elf"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	allowedELFTypeList = set.NewFrozenIntSet(int(elf.ET_EXEC), int(elf.ET_DYN))
)

// Metadata contains the exacted metadata from ELF file
type Metadata struct {
	// Sonames contains provided sonames for shared objects
	Sonames           []string
	ImportedLibraries []string
}

// GetExecutableMetadata extracts and returns Metadata if the input is an executable ELF binary.
// It is **not** an error if the passed in io.ReaderAt is not an ELF binary.
func GetExecutableMetadata(r io.ReaderAt) (*Metadata, error) {
	elfFile, err := elf.NewFile(r)
	if err != nil {
		// Do not return error if it is not in ELF format.
		if _, isFormatError := err.(*elf.FormatError); isFormatError {
			err = nil
		}
		return nil, err
	}
	defer utils.IgnoreError(elfFile.Close)

	// Exclude core and other unknown ELF file.
	if !allowedELFTypeList.Contains(int(elfFile.Type)) {
		return nil, nil
	}

	sonames, err := elfFile.DynString(elf.DT_SONAME)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sonames from ELF executable")
	}
	libraries, err := elfFile.ImportedLibraries()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get imported libraries from ELF executable")
	}
	return &Metadata{
		Sonames:           sonames,
		ImportedLibraries: libraries,
	}, nil
}
