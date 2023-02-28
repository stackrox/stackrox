package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	lock       sync.Mutex
	dataPosted = []string{}
)

func main() {
	go restServer()
	go syslog_server()
	select {}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()
	if len(dataPosted) == 0 {
		return
	}
	fmt.Fprintf(w, dataPosted[len(dataPosted)-1])
	dataPosted = dataPosted[:len(dataPosted)-1]
}

func restServer() {
	log.Println("Listening on localhost:8080.")
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Addr:              ":8080",
		Handler:           http.HandlerFunc(getHandler),
	}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func syslog_server() {
	log.Println("Listening on localhost:514.")
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:514")
	server.ListenTCP("0.0.0.0:514")
	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			fmt.Println(logParts)
			lock.Lock()
			dataPosted = append(dataPosted, fmt.Sprint(logParts))
			lock.Unlock()
		}
	}(channel)
	server.Wait()
}
