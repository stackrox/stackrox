package retry

// WithRetry allows you to call an error returning function with a suite of retry options and modifiers.
func WithRetry(f func() error, retriableOptions ...OptionsModifier) error {
	r := new(retryOptions)
	for _, modifier := range retriableOptions {
		modifier(r)
	}
	r.function = f
	return r.do()
}

// Tries adds a maximum number of attempts to make when retrying a function.
func Tries(n int) OptionsModifier {
	return func(o *retryOptions) { o.tries = n }
}

// OnlyRetryableErrors means only errors wrapped with MakeRetryable will be retried.
func OnlyRetryableErrors() OptionsModifier {
	return func(o *retryOptions) { o.canRetry = IsRetryable }
}

// OnFailedAttempts allows you to run a function on any failures, for instance logging failed attempts.
func OnFailedAttempts(onFailure func(error)) OptionsModifier {
	return func(o *retryOptions) { o.onFailure = onFailure }
}

// BetweenAttempts allows you to run any function in between different attempts, such as a backoff wait.
// BetweenAttempts and OnFailedAttempts are called at the same logical step, so you can use either or both.
func BetweenAttempts(between func()) OptionsModifier {
	return func(o *retryOptions) { o.between = between }
}

// OptionsModifier applies a mutation to a retryOptions.
type OptionsModifier func(*retryOptions)

type retryOptions struct {
	function  func() error
	onFailure func(error)
	canRetry  func(error) bool
	between   func()
	tries     int
}

func (t *retryOptions) do() (err error) {
	for i := 0; i < t.tries; i++ {
		// If we've run previously and have an error
		if err != nil {
			// Check if we can retry the error, and if so, run onFailure and between.
			if t.canRetry == nil || t.canRetry(err) {
				if t.onFailure != nil {
					t.onFailure(err)
				}
				if t.between != nil {
					t.between()
				}
			} else {
				// If we can't retry then return.
				return err
			}
		}

		// Try running the function. No error, no retry.
		if err = t.function(); err == nil {
			return
		}
	}
	return
}
