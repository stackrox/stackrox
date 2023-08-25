package config

// Config represents the Scanner configuration parameters.
type Config struct {
	// Mode specifies the mode in which Scanner will run.
	//
	// The options are: Indexer, Matcher, Combo.
	Mode string `json:"mode"`
}
