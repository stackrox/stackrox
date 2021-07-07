package auditlog

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/gogo/protobuf/types"
	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
)

var (
	log = logging.LoggerForModule()
)

type auditLogReaderImpl struct {
	logPath    string
	stopC      concurrency.Signal
	sender     auditLogSender
	startState *storage.AuditLogFileState
}

func (s *auditLogReaderImpl) StartReader(ctx context.Context) (bool, error) {
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

	go s.readAndForwardAuditLogs(ctx, t)
	return true, nil
}

func (s *auditLogReaderImpl) StopReader() bool {
	return s.stopC.Signal()
}

func (s *auditLogReaderImpl) readAndForwardAuditLogs(ctx context.Context, tailer *tail.Tail) {
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
					log.Errorf("Audit log tailing on file %s has unexpectedly ended: %v", s.logPath, err)
				} else {
					log.Errorf("Audit log tailing on file %s has unexpectedly ended", s.logPath)
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

			var auditLine auditEvent
			if err := json.Unmarshal([]byte(line.Text), &auditLine); err != nil {
				log.Errorf("Unable to parse log line: %v", err)
				continue // just move on
			}

			eventTS, err := auditLine.getEventTime()
			if err != nil {
				log.Errorf("Unable to parse timestamp from audit log: %v", err)
				continue
			}

			if !s.shouldSendEvent(&auditLine, eventTS) {
				continue
			}

			if err := s.sender.Send(ctx, &auditLine); err != nil {
				// It's very likely that this failure is due to Sensor being unavailable
				// In that case when Sensor next comes available it will ask to restart from the last
				// message it got. Therefore this event will end up being sent at that point.
				// Therefore, we skip retrying at this point in time.
				log.Errorf("Failed sending event to Sensor: %v", err)
				continue
			}
		}
	}
}

func (s *auditLogReaderImpl) shouldSendEvent(event *auditEvent, eventTS *types.Timestamp) bool {
	if s.startState != nil {
		if !protoutils.After(eventTS, s.startState.CollectLogsSince) {
			// don't send since time hasn't matched yet
			// but if the id matches then we're in the same time and everything after can be sent
			if event.AuditID == s.startState.LastAuditId {
				s.startState = nil
			}
			// in either case don't send
			return false
		}
		// otherwise the time is after prev state so clear out state and send (if it matches other conditions)
		s.startState = nil
	}

	return event.ObjectRef.Resource == "secrets" || event.ObjectRef.Resource == "configmaps"
}

func (s *auditLogReaderImpl) cleanupTailOnStop(tailer *tail.Tail) {
	if err := tailer.Stop(); err != nil {
		log.Errorf("Audit log tailer stopped with an error %v", err)
	}
	tailer.Cleanup()
}
