package schemas

// Registry allows retrieving schemas by name.
type Registry interface {
	GetSchema(name string) (Schema, error)
}
