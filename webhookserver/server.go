package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/stackrox/rox/pkg/sync"
)

type message struct {
	Headers map[string][]string    `json:"headers"`
	Data    map[string]interface{} `json:"data"`
}

var (
	lock       sync.Mutex
	dataPosted []message
)

func postHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	dataMap := make(map[string]interface{})
	if err := json.Unmarshal(data, &dataMap); err != nil {
		log.Printf("Error unmarshalling data: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	lock.Lock()
	defer lock.Unlock()
	dataPosted = append(dataPosted, message{Headers: r.Header, Data: dataMap})
	w.WriteHeader(http.StatusOK)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()
	resp, err := json.Marshal(&dataPosted)
	if err != nil {
		log.Printf("Failed to marshal resp: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write resp: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getHandler(w, r)
	case http.MethodPost:
		postHandler(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func tlsServer() {
	err := http.ListenAndServeTLS(":8443", "/tmp/certs/server.crt", "/tmp/certs/server.key", http.HandlerFunc(rootHandler))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func nonTLSServer() {
	if err := http.ListenAndServe(":8080", http.HandlerFunc(rootHandler)); err != nil {
		panic(err)
	}
}

func main() {
	go tlsServer()
	go nonTLSServer()
	select {}
}
