package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"strconv"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	utils "github.com/stackrox/rox/pkg/net"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/grpc"
)

const (
	parentUIDStr          = "ParentUid"
	parentExecFilePathStr = "ParentExecFilePath"
)

var (
	port                     = 9999
	dbPath                   = "/tmp/collector-test.db"
	processBucket            = "Process"
	networkBucket            = "Network"
	processLineageInfoBucket = "LineageInfo"
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

		fmt.Printf("Process: ")
		fmt.Printf("    name: %s\n", processSignal.GetName())
		fmt.Printf("    file-path: %s\n", processSignal.GetExecFilePath())
		fmt.Printf("    uid: %d\n", processSignal.GetUid())
		fmt.Printf("    gid: %d\n", processSignal.GetGid())
		fmt.Printf("    pid: %d\n", processSignal.GetPid())
		fmt.Printf("    args: %s\n", processSignal.GetArgs())

		processInfo := fmt.Sprintf("%s:%s:%d:%d:%d:%s", processSignal.GetName(), processSignal.GetExecFilePath(), processSignal.GetUid(), processSignal.GetGid(), processSignal.GetPid(), processSignal.GetArgs())
		fmt.Printf("ProcessInfo: %s %s\n", processSignal.GetContainerId(), processInfo)
		if err := s.UpdateProcessSignals(processSignal.GetContainerId(), processSignal.GetName(), processInfo); err != nil {
			return err
		}

		for _, info := range processSignal.GetLineageInfo() {
			processLineageInfo := fmt.Sprintf("%s:%s:%s:%d:%s:%s", processSignal.GetName(), processSignal.GetExecFilePath(), parentUIDStr, info.GetParentUid(), parentExecFilePathStr, info.GetParentExecFilePath())
			fmt.Printf("ProcessLineageInfo: %s %s\n", processSignal.GetContainerId(), processLineageInfo)

			id := fmt.Sprint(info.GetParentUid())
			if err := s.UpdateProcessLineageInfo(processSignal.GetName(), id, processLineageInfo); err != nil {
				return err
			}
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
			networkInfo := fmt.Sprintf("%s|%s|%s|%s|%s", getEndpoint(networkConn.GetLocalAddress()), getEndpoint(networkConn.GetRemoteAddress()), networkConn.GetRole().String(), networkConn.GetSocketFamily().String(), networkConn.GetCloseTimestamp().String())
			fmt.Printf("NetworkInfo: %s %s\n", networkConn.GetContainerId(), networkInfo)
			if err := s.UpdateNetworkConnInfo(networkConn.GetContainerId(), networkInfo); err != nil {
				return err
			}
		}

	}
}

func getEndpoint(networkAddress *sensorAPI.NetworkAddress) string {
	ipPortPair := utils.NetworkPeerID{
		Address: utils.IPFromBytes(networkAddress.GetAddressData()),
		Port:    uint16(networkAddress.GetPort()),
	}
	return ipPortPair.String()
}

func boltDB(path string) (db *bolt.DB, err error) {
	db, err = bolt.Open(path, 0777, nil)
	return db, err
}

func (s *signalServer) UpdateProcessSignals(containerID string, processName string, processInfo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(processBucket))
		if err != nil {
			return err
		}

		c, err := b.CreateBucketIfNotExists([]byte(containerID))
		if err != nil {
			return err
		}

		idx, err := c.NextSequence()
		if err != nil {
			return err
		}

		return c.Put([]byte(strconv.FormatUint(idx, 10)), []byte(processInfo))
	})
}

func (s *signalServer) UpdateProcessLineageInfo(processName string, parentID string, lineageInfo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, _ := tx.CreateBucketIfNotExists([]byte(processLineageInfoBucket))
		processBucket, _ := bucket.CreateBucketIfNotExists([]byte(processName))
		return processBucket.Put([]byte(parentID), []byte(lineageInfo))
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
