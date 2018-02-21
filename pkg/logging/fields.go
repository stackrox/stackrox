package logging

import "fmt"

//Fields provides the correctly typed input for the WithFields() method
type Fields map[string]interface{}

// String returns a string representation of the key/value map.
func (f Fields) String() string {
	str := ""

	for k, v := range f {
		str += fmt.Sprintf("\t%s: %s", k, v)
	}
	return str
}

// update returns a new fields map that is obtained by merging newFields into f (overwriting keys in f in case of
// clashes).
func (f Fields) update(newFields Fields) Fields {
	result := make(Fields, len(f)+len(newFields))
	for k, v := range f {
		result[k] = v
	}
	for k, v := range newFields {
		result[k] = v
	}
	return result
}
