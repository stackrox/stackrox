package caudit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type contextKey struct{}

type contextValue struct {
	events  []*Event
	eventMu sync.Mutex
}

type Status int

const (
	StatusSuccess Status = iota
	StatusFailure
)

func (s Status) String() string {
	switch s {
	case StatusSuccess:
		return "success"
	case StatusFailure:
		return "failure"
	default:
		return "unknown"
	}
}

type Event struct {
	Status  Status
	Time    time.Time
	Message string
}

func (e Event) String() string {
	ts := e.Time.Format(time.RFC3339)
	return fmt.Sprintf("[%s] %s: %s", ts, e.Status, e.Message)
}

func valueFromContext(ctx context.Context) *contextValue {
	v, ok := ctx.Value(contextKey{}).(*contextValue)
	if !ok {
		return nil
	}

	return v
}

// Events returns cloned events from the context.
func Events(ctx context.Context) []*Event {
	v := valueFromContext(ctx)

	v.eventMu.Lock()
	defer v.eventMu.Unlock()

	if v == nil {
		return nil
	}

	clonedEvents := make([]*Event, 0, len(v.events))
	for _, e := range v.events {
		clonedEvents = append(clonedEvents, &Event{
			Status:  e.Status,
			Time:    e.Time,
			Message: e.Message,
		})
	}

	return clonedEvents
}

// AddEvent is a noop if the context does not contain they key, otherwise
// adds an event to the context. Will return true if an event was added, false otherwise.
func AddEvent(ctx context.Context, status Status, format string, args ...any) bool {
	msg := fmt.Sprintf(format, args...)
	if ctx == nil {
		fmt.Printf("DAVE: ctx is nil - message skipped %q", msg)
		return false
	}

	v := valueFromContext(ctx)
	if v == nil {
		fmt.Printf("DAVE: valueFromContext is nil - message skipped %q", msg)
		return false
	}

	v.eventMu.Lock()
	defer v.eventMu.Unlock()

	v.events = append(v.events, &Event{
		Time:    time.Now(),
		Status:  status,
		Message: msg,
	})

	return true
}

func AddSuccessEvent(ctx context.Context, format string, args ...any) bool {
	return AddEvent(ctx, StatusSuccess, format, args...)
}

func AddFailureEvent(ctx context.Context, format string, args ...any) bool {
	return AddEvent(ctx, StatusFailure, format, args...)
}

func NewContext(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	return context.WithValue(parent, contextKey{}, &contextValue{})
}

// DebugDump dumps the audit contents to a string for printing / etc.
// Prefix will be prepended to each line to make it easier to grep logs.
func DebugDump(ctx context.Context, prefix string) string {
	sb := strings.Builder{}

	for _, e := range Events(ctx) {
		sb.WriteString(fmt.Sprintf("%s%s\n", prefix, e))
	}

	return sb.String()
}
