package globaldb

var (
	registeredTables = make(map[string]registeredTable)
)

type registeredTable struct {
	table, objType string
}

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(table string, objType string) {
	tableToRegister := registeredTable{
		table:   table,
		objType: objType,
	}

	if registered, ok := registeredTables[table]; ok {
		if registered != tableToRegister {
			log.Fatalf("table %q is already mapped to %q", table, registered.objType)
		}
		return
	}

	registeredTables[table] = tableToRegister
}
