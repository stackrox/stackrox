package recorder

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	flushTicker = 30 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// WrapStream either returns the stream wrapped with a recorder if it's a dev build and the variable is set
// or just returns the original stram
func WrapStream(server central.SensorService_CommunicateServer) central.SensorService_CommunicateServer {
	if !devbuild.IsEnabled() || env.RecorderTime.DurationSetting() == 0 {
		return server
	}
	log.Infof("[RECORDER] recorder is enabled for %0.2f minutes", env.RecorderTime.DurationSetting().Minutes())
	r := newRecordKeeper(server, env.RecorderTime.DurationSetting())
	return r
}

type record struct {
	nanos uint64
	msg   *central.MsgFromSensor
}

type recordKeeper struct {
	db *badger.DB
	central.SensorService_CommunicateServer
	recordDuration time.Duration

	started bool
	stopped concurrency.Signal

	queueMutex sync.Mutex
	queue      []record
}

func newRecordKeeper(server central.SensorService_CommunicateServer, recordDuration time.Duration) *recordKeeper {
	utils.Must(os.MkdirAll("/tmp/flightrecorder", 0777))

	db, err := badgerhelper.New("/tmp/flightrecorder", false)
	utils.Must(err)

	return &recordKeeper{
		db:                              db,
		SensorService_CommunicateServer: server,

		recordDuration: recordDuration,
		stopped:        concurrency.NewSignal(),
		queue:          make([]record, 0, 1000),
	}
}

func (r *recordKeeper) Recv() (*central.MsgFromSensor, error) {
	msg, err := r.SensorService_CommunicateServer.Recv()
	r.record(msg)
	return msg, err
}

func (r *recordKeeper) flush() {
	r.queueMutex.Lock()
	currentQueue := r.queue
	r.queue = make([]record, 0, 1000)
	r.queueMutex.Unlock()

	writeBatch := r.db.NewWriteBatch()
	for _, record := range currentQueue {
		data, err := proto.Marshal(record.msg)
		utils.Must(err)

		key := binenc.BigEndian.EncodeUint64(record.nanos)
		utils.Must(writeBatch.Set(key, data))
	}
	utils.Must(writeBatch.Flush())
}

func (r *recordKeeper) start(recordDuration time.Duration) {
	ticker := time.NewTicker(flushTicker)
	timer := time.NewTimer(recordDuration)
	for {
		select {
		case <-timer.C:
			r.stopped.Signal()
			ticker.Stop()

			r.flush()
			r.finish()
			return
		case <-ticker.C:
			r.flush()
		}
	}
}

func (r *recordKeeper) finish() {
	f, err := ioutil.TempFile("/tmp/flightrecorder", "")
	utils.Must(err)

	_, err = r.db.Backup(f, 0)
	utils.Must(err)

	utils.Must(r.db.Close())
	log.Infof("[RECORDER] wrote backup file to %q", f.Name())
}

func (r *recordKeeper) record(msg *central.MsgFromSensor) {
	if r.stopped.IsDone() {
		return
	}
	// Only start counting the events once the first message has been recv'd
	if !r.started {
		go r.start(r.recordDuration)
	}
	r.started = true

	r.queueMutex.Lock()
	defer r.queueMutex.Unlock()
	r.queue = append(r.queue, record{
		msg:   msg,
		nanos: uint64(time.Now().UnixNano()),
	})
}
