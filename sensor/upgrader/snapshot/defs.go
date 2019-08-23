package snapshot

const (
	// secretName is the name of the secret storing the upgrader state
	secretName = `sensor-upgrader-snapshot`
	// secretDataName is the key in the `data` map of the secret storing the gzip'd JSON data.
	secretDataName = `snapshot.json.gz`
)
