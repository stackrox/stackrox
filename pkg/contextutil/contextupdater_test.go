package contextutil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type testKey string

var (
	key1 = testKey("key1")
	key2 = testKey("key2")
)

func getValue(ctx context.Context, key testKey) string {
	s, _ := ctx.Value(key).(string)
	return s
}

func TestChainContextUpdaters_Success(t *testing.T) {
	a := assert.New(t)
	updater1ran := false
	updater2ran := false
	updater1 := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		a.False(updater1ran, "updater1 ran twice")
		a.False(updater2ran, "updater2 ran before updater1")
		newCtx := context.WithValue(ctx, key1, "value1")
		updater1ran = true
		return newCtx, nil
	})

	updater2 := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		a.True(updater1ran, "updater1 didn't run before updater2")
		a.False(updater2ran, "updater2 ran twice")
		a.Equal("value1", getValue(ctx, key1))
		newCtx := context.WithValue(ctx, key2, "value2")
		updater2ran = true
		return newCtx, nil
	})

	finalCtx, err := ChainContextUpdaters(updater1, updater2)(context.TODO())
	a.NoError(err)
	require.NotNil(t, finalCtx)
	a.True(updater1ran, "updater1 should have run")
	a.True(updater2ran, "updater1 should have run")

	a.Equal("value1", getValue(finalCtx, key1))
	a.Equal("value2", getValue(finalCtx, key2))
}

func TestChainContextUpdaters_Failure(t *testing.T) {
	a := assert.New(t)
	updater1ran := false
	updater1err := errors.New("error running updater1")
	updater1 := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		a.False(updater1ran, "updater1 ran twice")
		updater1ran = true
		return nil, updater1err
	})

	updater2 := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		a.FailNow("updater2 should not run")
		return nil, nil
	})

	_, err := ChainContextUpdaters(updater1, updater2)(context.TODO())
	a.Equal(updater1err, err)
}

func TestUnaryServerInterceptor_Success(t *testing.T) {
	a := assert.New(t)
	updater := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, key1, "value1"), nil
	})
	handler := grpc.UnaryHandler(func(ctx context.Context, req interface{}) (interface{}, error) {
		a.Equal("value1", getValue(ctx, key1))
		return "resp", nil
	})

	resp, err := UnaryServerInterceptor(updater)(context.TODO(), "req", nil, handler)
	a.NoError(err)
	a.Equal("resp", resp)
}

func TestUnaryServerInterceptor_FailUpdater(t *testing.T) {
	a := assert.New(t)
	updaterErr := errors.New("updater error")
	updater := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		return nil, updaterErr
	})
	handler := grpc.UnaryHandler(func(ctx context.Context, req interface{}) (interface{}, error) {
		a.FailNow("handler should not be called")
		return nil, nil
	})

	_, err := UnaryServerInterceptor(updater)(context.TODO(), "req", nil, handler)
	a.Equal(updaterErr, err)
}

func TestUnaryServerInterceptor_FailHandler(t *testing.T) {
	a := assert.New(t)
	handlerErr := errors.New("handler error")
	updater := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, key1, "value1"), nil
	})
	handler := grpc.UnaryHandler(func(ctx context.Context, req interface{}) (interface{}, error) {
		a.Equal("value1", getValue(ctx, key1))
		return nil, handlerErr
	})

	_, err := UnaryServerInterceptor(updater)(context.TODO(), "req", nil, handler)
	a.Equal(handlerErr, err)
}

type fakeStream struct {
	grpc.ServerStream
}

func (f fakeStream) Context() context.Context {
	return context.TODO()
}

func TestStreamServerInterceptor_Success(t *testing.T) {
	a := assert.New(t)
	updater := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, key1, "value1"), nil
	})
	handlerCalled := false
	handler := grpc.StreamHandler(func(srv interface{}, ss grpc.ServerStream) error {
		a.Equal("value1", getValue(ss.Context(), key1))
		handlerCalled = true
		return nil
	})

	err := StreamServerInterceptor(updater)("service", fakeStream{}, nil, handler)
	a.NoError(err)
	a.True(handlerCalled)
}

func TestStreamServerInterceptor_FailUpdater(t *testing.T) {
	a := assert.New(t)
	updaterErr := errors.New("updater error")
	updater := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		return nil, updaterErr
	})
	handler := grpc.StreamHandler(func(srv interface{}, ss grpc.ServerStream) error {
		a.FailNow("handler should not be called")
		return nil
	})

	err := StreamServerInterceptor(updater)("service", fakeStream{}, nil, handler)
	a.Equal(updaterErr, err)
}

func TestStreamServerInterceptor_FailHandler(t *testing.T) {
	a := assert.New(t)
	updater := ContextUpdater(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, key1, "value1"), nil
	})
	handlerErr := errors.New("handler error")
	handler := grpc.StreamHandler(func(srv interface{}, ss grpc.ServerStream) error {
		a.Equal("value1", getValue(ss.Context(), key1))
		return handlerErr
	})

	err := StreamServerInterceptor(updater)("service", fakeStream{}, nil, handler)
	a.Equal(handlerErr, err)
}
