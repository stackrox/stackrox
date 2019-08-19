package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

func recv(client central.SensorService_CommunicateClient) {
	for {
		if _, err := client.Recv(); err != nil {
			log.Error(err)
		}
	}
}

type record struct {
	nanos uint64
	msg   *central.MsgFromSensor
}

func main() {
	conn, err := clientconn.AuthenticatedGRPCConnection("central.stackrox:443", mtls.CentralSubject)
	utils.Must(err)

	client := central.NewSensorServiceClient(conn)

	stream, err := client.Communicate(context.Background())
	utils.Must(err)

	log.Infof("Connected to Central")

	_, err = stream.Header()
	utils.Must(err)

	_, err = stream.Recv()
	utils.Must(err)

	go recv(stream)

	utils.Must(os.MkdirAll("/tmp/recordreplay", 0777))
	db, err := badgerhelper.New("/tmp/recordreplay")
	utils.Must(err)

	f, err := os.Open("/recorder.db")
	utils.Must(err)

	err = db.Load(f, 10)
	utils.Must(err)

	log.Infof("Loaded replay DB")

	var records []record
	err = db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			val, err := it.Item().ValueCopy(nil)
			utils.Must(err)

			var msg central.MsgFromSensor
			utils.Must(proto.Unmarshal(val, &msg))

			records = append(records, record{
				nanos: binenc.BigEndian.Uint64(it.Item().Key()),
				msg:   &msg,
			})
		}
		return nil
	})
	utils.Must(err)

	log.Infof("Loaded %d records into memory", len(records))

	// Cut off the last event because what's one event
	for i := 0; i < len(records)-1; i++ {
		currRecord, nextRecord := records[i], records[i+1]

		go func() {
			err := stream.Send(currRecord.msg)
			utils.Must(err)
		}()

		time.Sleep(time.Duration(nextRecord.nanos - currRecord.nanos))
	}

	log.Info("Replay complete. Waiting till termination...")

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
}
