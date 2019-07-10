package renderer

// FileNameMap contains a set of source file names for rendering, optionally associating a target file name with each
// file.
// If the value for an entry in the map is empty, the target file name is assumed to be the base name of the source
// file.
type FileNameMap map[string]string

// NewFileNameMap returns a new FileNameMap with the given source files and no remapped target names.
func NewFileNameMap(files ...string) FileNameMap {
	return make(FileNameMap).Add(files...)
}

// AddWithName adds a source name with a remapped target file name.
func (m FileNameMap) AddWithName(srcName, tgtName string) FileNameMap {
	m[srcName] = tgtName
	return m
}

// Add adds the given source files to this FileNameMap, without remapping their target path.
func (m FileNameMap) Add(files ...string) FileNameMap {
	for _, file := range files {
		m[file] = ""
	}
	return m
}
