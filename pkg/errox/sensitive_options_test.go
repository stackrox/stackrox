package errox

import (
	"net"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestWithPublicMessage(t *testing.T) {
	err := &RoxSensitiveError{}
	WithPublicMessage("message")(err)
	assert.Equal(t, "message", err.public.Error())
	WithPublicMessage("another")(err)
	assert.Equal(t, "another: message", err.public.Error())
	assert.Nil(t, err.sensitive)
	assert.Empty(t, UnconcealSensitive(err))
}

func TestErrorIs(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		err := NewSensitive()
		assert.ErrorIs(t, err, nil)
	})
	t.Run("public", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithPublicError(NotFound)(err)
		assert.ErrorIs(t, err, NotFound)
		WithPublicError(InvalidArgs)(err)
		assert.ErrorIs(t, err, NotFound)
		WithPublicMessage("message")
		assert.ErrorIs(t, err, NotFound)
	})
	t.Run("sensitive", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithSensitive(testDNSError)(err)
		assert.ErrorIs(t, err, testDNSError)
		WithSensitive(InvalidArgs)(err)
		assert.ErrorIs(t, err, testDNSError)
	})
	t.Run("public sensitive", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithPublicError(NotFound)(err)
		WithSensitive(testDNSError)(err)
		assert.ErrorIs(t, err, testDNSError)
		assert.ErrorIs(t, err, NotFound)
	})
	t.Run("sensitive public", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithSensitive(testDNSError)(err)
		WithPublicError(NotFound)(err)
		assert.ErrorIs(t, err, testDNSError)
		assert.ErrorIs(t, err, NotFound)
	})
}

func TestWithSensitive(t *testing.T) {
	err := &RoxSensitiveError{}
	dnsError := &net.DNSError{Err: "DNS error", Name: "localhost", Server: "127.0.0.1"}
	WithSensitive(dnsError)(err)
	assert.Equal(t, "lookup: DNS error", err.Error())
	assert.Equal(t, "lookup: DNS error", err.sensitive.Error())
	assert.Equal(t, "lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))

	WithSensitive(errors.New("another"))(err)
	assert.Equal(t, "lookup: DNS error", err.Error())
	assert.Equal(t, "another: lookup: DNS error", err.sensitive.Error())
	assert.Equal(t, "another: lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))

	var serr error = MakeSensitive("public", errors.WithMessage(ConcealSensitive(dnsError), "secret"))
	err = &RoxSensitiveError{}
	WithSensitive(serr)(err)
	assert.Equal(t, "public: lookup: DNS error", err.Error())
	assert.Equal(t, "public: lookup: DNS error", err.sensitive.Error())
	assert.Equal(t, "secret: lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))

	serr = errors.WithMessage(ConcealSensitive(dnsError), "secret")
	err = &RoxSensitiveError{}
	WithSensitive(serr)(err) // adds dns public message to the public part
	assert.Equal(t, "lookup: DNS error", err.Error())
	assert.Equal(t, "secret: lookup: DNS error", err.sensitive.Error())
	assert.Equal(t, "secret: lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))
}

func TestWithSensitivef(t *testing.T) {
	err := &RoxSensitiveError{}
	WithSensitivef("format %v", "value")(err)
	assert.Nil(t, err.public)
	assert.Empty(t, err.Error())
	assert.Equal(t, "format value", err.sensitive.Error())
	assert.Equal(t, err.sensitive.Error(), UnconcealSensitive(err))

	err = &RoxSensitiveError{}
	WithPublicMessage("public")(err)
	WithSensitivef("format %v", "value")(err)
	assert.Equal(t, "public", err.public.Error())
	assert.Equal(t, "format value: public", err.sensitive.Error())
	assert.Equal(t, "public", err.Error())
	assert.Equal(t, err.sensitive.Error(), UnconcealSensitive(err))
}

func TestWithPublicError(t *testing.T) {
	err := &RoxSensitiveError{}
	WithPublicError(errors.New("public"))(err)
	assert.Equal(t, "public", err.public.Error())
	assert.Equal(t, err.public.Error(), err.Error())
	assert.Nil(t, err.sensitive)

	WithPublicError(errors.New("another"))(err)
	assert.Equal(t, "another: public", err.public.Error())
	assert.Equal(t, err.public.Error(), err.Error())
	assert.Nil(t, err.sensitive)
}

func TestOrder(t *testing.T) {
	t.Run("public message, sensitive", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithPublicMessage("message")(err)
		WithSensitive(testDNSError)(err)

		assert.Equal(t, "message", err.public.Error())
		assert.Equal(t, "message: lookup: DNS error", err.Error())
		assert.Equal(t, "lookup: DNS error", err.sensitive.Error())
		assert.Equal(t, "lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))
	})
	t.Run("sensitive, public message", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithSensitive(testDNSError)(err)
		WithPublicMessage("message")(err)

		assert.Equal(t, "message", err.public.Error())
		assert.Equal(t, "message: lookup: DNS error", err.Error())
		assert.Equal(t, "lookup: DNS error", err.sensitive.Error())
		assert.Equal(t, "lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))
	})

	public := errors.New("message")
	t.Run("public error, sensitive", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithPublicError(public)(err)
		WithSensitive(testDNSError)(err)

		assert.Equal(t, "message", err.public.Error())
		assert.Equal(t, "message: lookup: DNS error", err.Error())
		assert.Equal(t, "lookup: DNS error", err.sensitive.Error())
		assert.Equal(t, "lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))
	})
	t.Run("sensitive, public error", func(t *testing.T) {
		err := &RoxSensitiveError{}
		WithSensitive(testDNSError)(err)
		WithPublicError(public)(err)

		assert.Equal(t, "message", err.public.Error())
		assert.Equal(t, "message: lookup: DNS error", err.Error())
		assert.Equal(t, "lookup: DNS error", err.sensitive.Error())
		assert.Equal(t, "lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))
	})
}

func TestNewSensitive(t *testing.T) {
	err := NewSensitive()
	assert.Nil(t, err)

	public := errors.New("message")
	err = NewSensitive(WithPublicError(public))
	assert.Equal(t, public, err)

	err = NewSensitive(WithSensitive(testDNSError), WithPublicError(public))
	assert.Equal(t, "message: lookup: DNS error", err.Error())
	assert.Equal(t, "lookup localhost on 127.0.0.1: DNS error", UnconcealSensitive(err))
}
