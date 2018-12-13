package main

import (
	"fmt"
	"log"
	"net"

	bolt "github.com/etcd-io/bbolt"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/grpc"
)

var (
	port          = 9999
	dbPath        = "/tmp/collector-test.db"
	processBucket = "Process"
)

type signalServer struct {
	db *bolt.DB
}

func newServer(db *bolt.DB) *signalServer {
	return &signalServer{
		db: db,
	}
}

func (s *signalServer) PushSignals(stream sensorAPI.SignalService_PushSignalsServer) error {
	for {
		signal, err := stream.Recv()
		if err != nil {
			return err
		}
		var processSignal *storage.ProcessSignal
		if signal != nil && signal.GetSignal() != nil && signal.GetSignal().GetProcessSignal() != nil {
			processSignal = signal.GetSignal().GetProcessSignal()
		}

		fmt.Printf("%v\n", signal.GetSignal().GetProcessSignal())
		if err := s.Update(processSignal); err != nil {
			return err
		}
	}
}

func boltDB(path string) (db *bolt.DB, err error) {
	db, err = bolt.Open(path, 0777, nil)
	return db, err
}

func (s *signalServer) Update(processSignal *storage.ProcessSignal) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(processBucket))
		return b.Put([]byte(processSignal.Name), []byte(processSignal.ExecFilePath))
	})
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	db, err := boltDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	grpcServer := grpc.NewServer()
	sensorAPI.RegisterSignalServiceServer(grpcServer, newServer(db))
	grpcServer.Serve(lis)
}
