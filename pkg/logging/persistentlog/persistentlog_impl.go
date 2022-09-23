package persistentlog

import (
	"context"
	"io"
	"os"

	"github.com/gogo/protobuf/types"
	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/persistentlog/store"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()

	// stagesAllowList is the set of stages that will be sent.
	// Currently only `ResponseComplete` ("The response body has been completed, and no more bytes will be sent.") OR
	// `Panic` ("Events generated when a panic occurred.") are supported
	stagesAllowList = set.NewFrozenStringSet("ResponseComplete", "Panic")

	// resourceTypesAllowList is set of resources that will be sent.
	resourceTypesAllowList = set.NewFrozenStringSet("secrets", "configmaps")

	// verbsDenyList is the set of verbs that will NOT be sent if encountered.
	verbsDenyList = set.NewFrozenStringSet("WATCH", "LIST")
)

type persistentLogReaderImpl struct {
	logPath            string
	stopC              concurrency.Signal
	persistentLogStore store.Store
}

func (s *persistentLogReaderImpl) StartReader(ctx context.Context) (bool, error) {
	if s.stopC.IsDone() {
		return false, errors.New("Cannot start reader because stopC is already done. Reader might have already been stopped")
	}

	t, err := tail.TailFile(s.logPath, tail.Config{
		ReOpen:    true,
		MustExist: true,
		Follow:    true,
	})

	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Audit log file %s doesn't exist on this compliance node", s.logPath)
			return false, nil
		}
		// handle other errors
		log.Errorf("Failed to open file: %v", err)
		return false, err
	}

	go s.readAndStorePersistentLogs(ctx, t)
	return true, nil
}

func (s *persistentLogReaderImpl) StopReader() bool {
	return s.stopC.Signal()
}

func (s *persistentLogReaderImpl) readAndStorePersistentLogs(ctx context.Context, tailer *tail.Tail) {
	defer s.cleanupTailOnStop(tailer)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopC.Done():
			return
		case line := <-tailer.Lines:
			if line == nil {
				err := tailer.Err()
				// For some reason the channel has been closed. We don't expect to ever get any more data here so return
				if err != nil {
					log.Errorf("Persistent log tailing on file %s has unexpectedly ended: %v", s.logPath, err)
				} else {
					log.Errorf("Persistent log tailing on file %s has unexpectedly ended", s.logPath)
				}
				// TODO: report health failure
				return
			}

			if line.Err != nil {
				if line.Err == io.EOF {
					continue // Ignore. When file gets rotated or more data comes in we will get next line
				}
				log.Errorf("Failed tailing log: %v", line.Err)
				// TODO: report health failure
				return
			}

			logData := storage.PersistentLog{
				Log:       line.Text,
				Timestamp: types.TimestampNow(),
			}
			if err := s.persistentLogStore.Upsert(ctx, &logData); err != nil {
				log.Errorf("Unable to parse log line: %v", err)
				continue // just move on
			}
		}
	}
}

func (s *persistentLogReaderImpl) cleanupTailOnStop(tailer *tail.Tail) {
	// Only want to call tailer.Stop here as that stops the tailing. However do _not_ call tailer.Cleanup() because as the docs mention
	// "If you plan to re-read a file, don't call Cleanup in between."
	// Cleanup is recommended for after the process exits, but that's not strictly necessary as if the process exits the container will be restarted anyway
	// See https://pkg.go.dev/github.com/nxadm/tail#Tail.Cleanup for details on Cleanup()
	if err := tailer.Stop(); err != nil {
		log.Errorf("Persistent log tailer stopped with an error %v", err)
	}
}
