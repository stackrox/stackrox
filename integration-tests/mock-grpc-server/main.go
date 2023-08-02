package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	utils "github.com/stackrox/rox/pkg/net"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
	endpointBucket           = "Endpoint"
	processLineageInfoBucket = "LineageInfo"
)

type signalServer struct {
	sensorAPI.UnimplementedSignalServiceServer
	sensorAPI.UnimplementedNetworkConnectionInfoServiceServer

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
		} else {
			continue
		}

		processInfo := fmt.Sprintf("%s:%s:%d:%d:%d:%s", processSignal.GetName(), processSignal.GetExecFilePath(), processSignal.GetUid(), processSignal.GetGid(), processSignal.GetPid(), processSignal.GetArgs())
		fmt.Printf("ProcessInfo: %s %s\n", processSignal.GetContainerId(), processInfo)
		if err := s.UpdateBucket(processSignal.GetContainerId(), processInfo, processBucket); err != nil {
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
		networkEndpoints := networkConnInfo.GetUpdatedEndpoints()

		for _, networkConn := range networkConns {
			networkInfo := fmt.Sprintf("%s|%s|%s|%s|%s", getEndpoint(networkConn.GetLocalAddress()), getEndpoint(networkConn.GetRemoteAddress()), networkConn.GetRole().String(), networkConn.GetSocketFamily().String(), networkConn.GetCloseTimestamp().String())
			fmt.Printf("NetworkInfo: %s %s\n", networkConn.GetContainerId(), networkInfo)
			if err := s.UpdateBucket(networkConn.GetContainerId(), networkInfo, networkBucket); err != nil {
				return err
			}
		}

		for _, networkEndpoint := range networkEndpoints {
			endpointInfo := fmt.Sprintf("EndpointInfo: %s|%s|%s|%s|%s\n", networkEndpoint.GetSocketFamily().String(), networkEndpoint.GetProtocol().String(), networkEndpoint.GetListenAddress().String(), networkEndpoint.GetCloseTimestamp().String(), networkEndpoint.GetOriginator().String())
			fmt.Printf("EndpointInfo: %s %s\n", networkEndpoint.GetContainerId(), endpointInfo)
			if err := s.UpdateBucket(networkEndpoint.GetContainerId(), endpointInfo, endpointBucket); err != nil {
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

func (s *signalServer) UpdateProcessLineageInfo(processName string, parentID string, lineageInfo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, _ := tx.CreateBucketIfNotExists([]byte(processLineageInfoBucket))
		processBucket, _ := bucket.CreateBucketIfNotExists([]byte(processName))
		return processBucket.Put([]byte(parentID), []byte(lineageInfo))
	})
}

func (s *signalServer) UpdateBucket(containerID string, info string, bucket string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
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

		return c.Put([]byte(strconv.FormatUint(idx, 10)), []byte(info))
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

	maxMsgSize := 12 * 1024 * 1024
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: 40 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	sensorAPI.RegisterSignalServiceServer(grpcServer, newServer(db))
	sensorAPI.RegisterNetworkConnectionInfoServiceServer(grpcServer, newServer(db))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// listening OS shutdown signal
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
