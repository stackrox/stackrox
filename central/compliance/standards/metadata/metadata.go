package metadata

var (
	// AllStandards is the list of all compliance standards.
	AllStandards []Standard
)

// Control is the metadata of a control in a compliance standard.
type Control struct {
	ID   string
	Name string

	Description string
}

// Category represents a category in a compliance standard.
type Category struct {
	ID          string
	Name        string
	Description string

	Controls []Control
}

// Standard represents a compliance standard.
type Standard struct {
	ID          string
	Name        string
	Description string
	Dynamic     bool

	Categories []Category
}
