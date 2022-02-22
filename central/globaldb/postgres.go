package globaldb

var (
	registeredTables = make(map[string]registeredTable)
)

type registeredTable struct {
	table, objType string
}

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(table string, objType string) {
	if registered, ok := registeredTables[table]; ok {
		log.Fatalf("table %q is already mapped to %q", table, registered.objType)
	}
	registeredTables[table] = registeredTable{
		table:   table,
		objType: objType,
	}
}
