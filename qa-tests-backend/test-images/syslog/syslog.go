package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"gopkg.in/mcuadros/go-syslog.v2"
)

var (
	lock       sync.Mutex
	dataPosted = []string{}
)

func main() {
	go restServer()
	go syslogServer()
	select {}
}

func getHandler(w http.ResponseWriter, _ *http.Request) {
	lock.Lock()
	defer lock.Unlock()
	if len(dataPosted) == 0 {
		return
	}
	fmt.Fprint(w, dataPosted[len(dataPosted)-1])
	dataPosted = dataPosted[:len(dataPosted)-1]
}

func restServer() {
	log.Println("Listening on localhost:8080")
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Addr:              ":8080",
		Handler:           http.HandlerFunc(getHandler),
	}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func syslogServer() {
	log.Println("Listening on localhost:514")
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	if err := server.ListenUDP("0.0.0.0:514"); err != nil {
		panic(err)
	}
	if err := server.ListenTCP("0.0.0.0:514"); err != nil {
		panic(err)
	}
	if err := server.Boot(); err != nil {
		panic(err)
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			log.Println(logParts)
			concurrency.WithLock(&lock, func() {
				dataPosted = append(dataPosted, fmt.Sprint(logParts))
			})
		}
	}(channel)
	server.Wait()
}
