package errox

import (
	"net"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestSensitiveError(t *testing.T) {
	secret := errors.New("SECRET")

	t.Run("concealed secret", func(t *testing.T) {
		var sensitive error = MakeSensitive("******", secret)
		var serr SensitiveError

		assert.Equal(t, "******", sensitive.Error())
		assert.ErrorAs(t, sensitive, &serr)
	})
	t.Run("GetSensitiveError", func(t *testing.T) {
		var sensitive error = MakeSensitive("******", secret)
		wrapped := errors.Wrap(sensitive, "message")

		assert.Contains(t, GetSensitiveError(sensitive), "SECRET")
		assert.Equal(t, "message: ******", wrapped.Error())
		assert.Equal(t, "message: SECRET", GetSensitiveError(wrapped))
	})
	t.Run("triple wrap", func(t *testing.T) {
		err := errors.New("subsecret")
		var sensitive error = MakeSensitive("******", errors.Wrap(err, "SECRET"))
		wrapped := errors.Wrap(sensitive, "message")

		assert.Equal(t, "message: ******", wrapped.Error())
		assert.Equal(t, "message: SECRET: subsecret", GetSensitiveError(wrapped))
	})
	t.Run("triple wrap WithMessage", func(t *testing.T) {
		err := errors.New("subsecret")
		var sensitive error = MakeSensitive("******", errors.WithMessage(err, "SECRET"))
		wrapped := errors.WithMessage(sensitive, "message")

		assert.Equal(t, "message: ******", wrapped.Error())
		assert.Equal(t, "message: SECRET: subsecret", GetSensitiveError(wrapped))
	})
	t.Run("two sensitives in a chain", func(t *testing.T) {
		wrapped := errors.WithMessage(
			MakeSensitive("******",
				errors.WithMessage(
					errors.WithMessage(
						MakeSensitive("!!!!!!",
							errors.WithMessage(
								errors.New("subsecret again"),
								"SECOND")),
						"subsecret"),
					"FIRST")),
			"message")

		assert.Equal(t, "message: ******: !!!!!!", wrapped.Error())
		assert.Equal(t, "message: FIRST: subsecret: SECOND: subsecret again", GetSensitiveError(wrapped))
	})
	t.Run("nil", func(t *testing.T) {
		var sensitive error = MakeSensitive("******", nil)
		var serr SensitiveError

		assert.Equal(t, "******", sensitive.Error())
		assert.ErrorAs(t, sensitive, &serr)
		assert.NotNil(t, serr)
	})
	t.Run("not sensitive", func(t *testing.T) {
		assert.Equal(t, "SECRET", GetSensitiveError(secret))
		assert.Equal(t, "message: SECRET", GetSensitiveError(errors.Wrap(secret, "message")))
	})
	t.Run("async", func(t *testing.T) {
		sensitive := MakeSensitive("******", secret)
		sensitive.unprotect()
		var message string
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			message = sensitive.Error()
			wg.Done()
		}()
		sensitive.protect()
		wg.Wait()
		assert.NotContains(t, message, "SECRET")
		assert.Contains(t, GetSensitiveError(sensitive), "SECRET")
		assert.Contains(t, message, "******")
	})
}

func TestConsealSensitive(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		err := ConsealSensitive(nil)
		assert.Nil(t, err)
	})
	t.Run("not sensitive", func(t *testing.T) {
		err := errors.New("non-sensitive")
		assert.Equal(t, err, ConsealSensitive(err))
	})
	t.Run("DNSError", func(t *testing.T) {
		err := errors.Wrap(&net.DNSError{Name: "hehe", Server: "1.2.3.4", Err: "oops"}, "message")
		err = ConsealSensitive(err)
		assert.Equal(t, "lookup: oops", err.Error())
		assert.Equal(t, "message: lookup hehe on 1.2.3.4: oops", GetSensitiveError(err))
	})
	t.Run("OpError", func(t *testing.T) {
		err := errors.Wrap(&net.OpError{Op: "dial", Net: "tcp",
			Source: &net.TCPAddr{IP: net.IPv4(1, 2, 3, 4)}, Addr: &net.TCPAddr{IP: net.IPv4(5, 6, 7, 8)},
			Err: errors.New("oops")}, "message")
		err = ConsealSensitive(err)
		assert.Equal(t, "dial tcp: oops", err.Error())
		assert.Equal(t, "message: dial tcp 1.2.3.4:0->5.6.7.8:0: oops", GetSensitiveError(err))
	})
}

func Test_isProtected(t *testing.T) {
	t.Run("protected", func(t *testing.T) {
		sensitive := MakeSensitive("non-secret", NotImplemented)
		assert.True(t, sensitive.isProtected())
		wg := sync.WaitGroup{}
		wg.Add(1)
		var protected bool
		go func() {
			protected = sensitive.isProtected()
			wg.Done()
		}()
		wg.Wait()
		assert.True(t, protected)
	})
	t.Run("protected in a parallel goroutine", func(t *testing.T) {
		sensitive := MakeSensitive("non-secret", NotImplemented)
		sensitive.unprotect()
		defer sensitive.protect()
		wg := sync.WaitGroup{}
		wg.Add(1)
		var protected bool
		var id int
		go func() {
			id = getGoroutineID()
			protected = sensitive.isProtected()
			wg.Done()
		}()
		wg.Wait()
		assert.Equal(t, getGoroutineID(), int(sensitive.unprotectedGoRoutineID.Load()))
		assert.NotEqual(t, id, int(sensitive.unprotectedGoRoutineID.Load()))
		assert.True(t, protected, "must be protected in another goroutine")
		assert.False(t, sensitive.isProtected(), "must be unprotected in the current")
	})
}

func Test_getGoroutineID(t *testing.T) {
	current := getGoroutineID()
	another := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		another = getGoroutineID()
		wg.Done()
	}()
	wg.Wait()
	assert.NotEqual(t, 0, current)
	assert.NotEqual(t, 0, another)
	assert.NotEqual(t, current, another)
}
