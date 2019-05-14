package flags

import (
	"fmt"
	"io/ioutil"
)

// FileContentsVar can be used for a flag that takes a file name, and reads the raw contents
// from the given file.
type FileContentsVar struct {
	Filename string
	Data     *[]byte
}

// Type implements the value interface.
func (FileContentsVar) Type() string {
	return "file"
}

// String implements the value interface
func (v FileContentsVar) String() string {
	if v.Data == nil || v.Filename == "" {
		return ""
	}
	return fmt.Sprintf("<contents of file %s>", v.Filename)
}

// Set implements the value interface.
func (v *FileContentsVar) Set(val string) error {
	if val == "" {
		if v.Data != nil {
			*v.Data = nil
		}
		v.Filename = ""
		return nil
	}

	var err error
	*v.Data, err = ioutil.ReadFile(val)
	if err != nil {
		return err
	}

	v.Filename = val
	return nil
}
