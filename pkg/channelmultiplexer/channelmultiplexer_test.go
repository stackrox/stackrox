package channelmultiplexer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type msgstruct struct {
	i   int
	str string
}

func TestMultiplexingChannels(t *testing.T) {
	chans := make([]chan string, 5)
	for i := range chans {
		chans[i] = make(chan string)
		defer close(chans[i])
	}
	cases := map[string]struct {
		inputChannels []chan string
		messages      []msgstruct
	}{
		"No input": {
			inputChannels: nil,
			messages:      nil,
		},
		"Single message": {
			inputChannels: []chan string{chans[0]},
			messages:      []msgstruct{{i: 0, str: "First message"}},
		},
		"Single channel": {
			inputChannels: []chan string{chans[1]},
			messages:      []msgstruct{{i: 0, str: "First message"}, {i: 0, str: "Second message"}, {i: 0, str: "Third message"}},
		},
		"Multiple channels": {
			inputChannels: chans[2:],
			messages:      []msgstruct{{i: 0, str: "First message"}, {i: 2, str: "Second message"}, {i: 1, str: "Third message"}, {i: 0, str: "Fourth message"}},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			mp := NewMultiplexer[string]()
			for _, ch := range c.inputChannels {
				mp.AddChannel(ch)
			}

			mp.Run()

			for _, m := range c.messages {
				c.inputChannels[m.i] <- m.str
				received, ok := <-mp.GetOutput()
				assert.True(t, ok)
				assert.Equal(t, m.str, received)
			}
		})
	}
}

func TestCancelContext(t *testing.T) {
	numChannels := 2
	ctx, cancelFn := context.WithCancel(context.Background())
	cm := NewMultiplexer[int](WithContext[int](ctx))
	channels := make([]chan int, numChannels)
	for i := range channels {
		channels[i] = make(chan int)
		cm.AddChannel(channels[i])
	}
	t.Cleanup(func() {
		for _, c := range channels {
			close(c)
		}
	})

	cm.Run()
	cancelFn()

	select {
	case <-time.NewTimer(5 * time.Second).C:
		t.Error("timeout reached waiting for the channel multiplexer to stop")
	case _, ok := <-cm.GetOutput():
		assert.False(t, ok, "the output channel should be closed")
	}
}
