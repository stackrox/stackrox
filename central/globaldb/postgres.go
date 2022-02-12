package globaldb

var (
	registeredTables []registeredTable
)

type registeredTable struct {
	table, objType string
}

func RegisterTable(table string, objType string) {
	registeredTables = append(registeredTables, registeredTable{
		table:   table,
		objType: objType,
	})
}
