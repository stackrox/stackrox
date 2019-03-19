package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
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
	data, err := ioutil.ReadAll(r.Body)
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

func main() {
	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
