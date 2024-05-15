package message

// AcscsEmail is the type used to send a message via an acscsemail.Client
type AcscsEmail struct {
	To         []string `json:"to"`
	RawMessage []byte   `json:"rawMessage"`
}
