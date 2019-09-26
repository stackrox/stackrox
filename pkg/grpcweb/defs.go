package grpcweb

const (
	compressedFlag     byte = 1 << 0
	trailerMessageFlag byte = 1 << 7

	completeHeaderLen = 5
)
