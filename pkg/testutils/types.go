package testutils

// T generalizes testing.T
type T interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	FailNow()
	Logf(format string, args ...interface{})
}
