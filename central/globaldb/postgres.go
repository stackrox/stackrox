package globaldb

var (
	registeredTables []registeredTable
)

type registeredTable struct {
	table, objType string
}

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(table string, objType string) {
	registeredTables = append(registeredTables, registeredTable{
		table:   table,
		objType: objType,
	})
}
