package rate

type Payload struct {
	QueueName  string
	QueueSize  int
	NumDropped int
}
