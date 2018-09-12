package retry

// MakeRetryable is an explicit wrapper for errors you want to retry if you use the IsRetryable function with
// the OnlyIf option.
func MakeRetryable(e error) error {
	if e == nil {
		panic("retiryable is an error type, nil is no error at all.")
	}
	return &retryableError{error: e}
}

type retryableError struct {
	error
}

// IsRetryable returns if the error is an instance of RetryableError
func IsRetryable(e error) bool {
	_, ir := e.(*retryableError)
	return ir
}
