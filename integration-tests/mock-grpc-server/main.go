package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	bolt "github.com/etcd-io/bbolt"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	utils "github.com/stackrox/rox/pkg/net"
	"google.golang.org/grpc"
)

var (
	port          = 9999
	dbPath        = "/tmp/collector-test.db"
	processBucket = "Process"
	networkBucket = "Network"
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

		processInfo := fmt.Sprintf("%s:%s:%d:%d", processSignal.GetName(), processSignal.GetExecFilePath(), processSignal.GetUid(), processSignal.GetGid())
		fmt.Printf("ProcessInfo: %s %s\n", processSignal.GetContainerId(), processInfo)
		if err := s.UpdateProcessSignals(processSignal.GetName(), processInfo); err != nil {
			return err
		}
	}
}

func (s *signalServer) PushNetworkConnectionInfo(stream sensorAPI.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	for {
		signal, err := stream.Recv()
		if err != nil {
			fmt.Println(err)
			return err
		}
		networkConnInfo := signal.GetInfo()
		networkConns := networkConnInfo.GetUpdatedConnections()

		for _, networkConn := range networkConns {
			networkInfo := fmt.Sprintf("%s:%s:%s:%s", getEndpoint(networkConn.GetLocalAddress()), getEndpoint(networkConn.GetRemoteAddress()), networkConn.GetRole().String(), networkConn.GetSocketFamily().String())
			fmt.Printf("NetworkInfo: %s %s\n", networkConn.GetContainerId(), networkInfo)
			if err := s.UpdateNetworkConnInfo(networkConn.GetContainerId(), networkInfo); err != nil {
				return err
			}
		}

	}
}

func getEndpoint(networkAddress *sensorAPI.NetworkAddress) string {
	ipPortPair := utils.IPPortPair{
		Address: utils.IPFromBytes(networkAddress.GetAddressData()),
		Port:    uint16(networkAddress.GetPort()),
	}
	return ipPortPair.String()
}

func boltDB(path string) (db *bolt.DB, err error) {
	db, err = bolt.Open(path, 0777, nil)
	return db, err
}

func (s *signalServer) UpdateProcessSignals(processName string, processInfo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(processBucket))
		return b.Put([]byte(processName), []byte(processInfo))
	})
}

func (s *signalServer) UpdateNetworkConnInfo(containerID string, networkInfo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(networkBucket))

		err := b.Put([]byte(containerID), []byte(networkInfo))
		if err != nil {
			fmt.Println(err)
			return err
		}

		return nil
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

	grpcServer := grpc.NewServer()
	sensorAPI.RegisterSignalServiceServer(grpcServer, newServer(db))
	sensorAPI.RegisterNetworkConnectionInfoServiceServer(grpcServer, newServer(db))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	fmt.Println("Got OS shutdown signal, shutting down grpc server gracefully...")
	grpcServer.Stop()
	err = db.Close()
	// Db not being closed properly affects test
	if err != nil {
		fmt.Println(err)
	}
}
