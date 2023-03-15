package channelmultiplexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type msgstruct struct {
	i   int
	str string
}

func TestMultiplexingChannels(t *testing.T) {
	cases := map[string]struct {
		inputChannels []chan *string
		messages      []msgstruct
	}{
		"No input": {
			inputChannels: nil,
			messages:      nil,
		},
		"Single message": {
			inputChannels: []chan *string{make(chan *string)},
			messages:      []msgstruct{{i: 0, str: "First message"}},
		},
		"Single channel": {
			inputChannels: []chan *string{make(chan *string)},
			messages:      []msgstruct{{i: 0, str: "First message"}, {i: 0, str: "Second message"}, {i: 0, str: "Third message"}},
		},
		"Multiple channels": {
			inputChannels: []chan *string{make(chan *string), make(chan *string), make(chan *string)},
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
				c.inputChannels[m.i] <- &m.str
				received, ok := <-mp.GetOutput()
				assert.True(t, ok)
				assert.Equal(t, m.str, *received)
			}
		})
	}
}
