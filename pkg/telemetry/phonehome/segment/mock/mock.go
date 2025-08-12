package mock

import (
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
)

type segmentMessageTemplate struct {
	Batch []map[string]any `json:"batch"`
}

// FilterMessageFields returns an iterator over existing message fields and
// their values.
func FilterMessageFields(message map[string]any, fields ...string) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for _, field := range fields {
			if value, ok := message[field]; ok {
				if !yield(field, value) {
					break
				}
			}
		}
	}
}

// NewServer returns an instance of a HTTP server and a data channel.
// The server expects JSON messages in the following format:
//
//	{
//	  "batch": [
//	    {
//	      // map[string]any
//	    }
//	  ]
//	}
//
// The returned data channel will receive parsed messages.
func NewServer(buffer int) (*httptest.Server, <-chan map[string]any) {
	dataCh := make(chan map[string]any, buffer)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		d := json.NewDecoder(r.Body)
		var message segmentMessageTemplate
		if d.Decode(&message) == nil {
			for _, m := range message.Batch {
				dataCh <- m
			}
		}
	}))
	server.Config.RegisterOnShutdown(func() {
		close(dataCh)
	})
	server.Start()
	return server, dataCh
}
