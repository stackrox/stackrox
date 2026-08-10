package liveprobe

type ServiceDef struct {
	Name     string
	Port     int
	Protocol string // "tls" or "postgres"
}

var CentralServices = []ServiceDef{
	{Name: "central", Port: 443, Protocol: "tls"},
	{Name: "central-db", Port: 5432, Protocol: "postgres"},
	{Name: "scanner", Port: 8443, Protocol: "tls"},
	{Name: "scanner-db", Port: 5432, Protocol: "postgres"},
	{Name: "scanner-v4-indexer", Port: 8443, Protocol: "tls"},
	{Name: "scanner-v4-matcher", Port: 8443, Protocol: "tls"},
	{Name: "scanner-v4-db", Port: 5432, Protocol: "postgres"},
}

var SecuredClusterServices = []ServiceDef{
	{Name: "sensor", Port: 443, Protocol: "tls"},
	{Name: "admission-control", Port: 443, Protocol: "tls"},
	{Name: "scanner", Port: 8443, Protocol: "tls"},
	{Name: "scanner-db", Port: 5432, Protocol: "postgres"},
	{Name: "scanner-v4-indexer", Port: 8443, Protocol: "tls"},
	{Name: "scanner-v4-db", Port: 5432, Protocol: "postgres"},
}
