package auditlog

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type auditLogReaderImpl struct {
	logPath string
	stopC   concurrency.Signal
	sender  auditLogSender
}

func (s *auditLogReaderImpl) StartReader(ctx context.Context) (bool, error) {
	if s.stopC.IsDone() {
		return false, errors.New("Cannot start reader because stopC is already done. Reader might have already been stopped")
	}

	//TODO: Restart processing based on previous state: https://stack-rox.atlassian.net/browse/ROX-7175
	t, err := tail.TailFile(s.logPath, tail.Config{
		ReOpen:    true,
		MustExist: true,
		Follow:    true,
	})

	if err != nil {
		if os.IsNotExist(err) {
			// TODO: Only gracefully exit if this is _not_ a master node. This can be done once that information is sent from Sensor
			// (or once senor only starts the process on compliance running on master nodes)
			// gracefully exit if this is not on master nodes and hence doesn't have the k8s audit logs
			log.Infof("Audit log file %s doesn't exist on this compliance node", s.logPath)
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

			if s.shouldSendEvent(&auditLine) {
				if err := s.sender.Send(ctx, &auditLine); err != nil {
					log.Errorf("Failed sending event to Sensor: %v", line.Err)
					// TODO: report health, etc
				}
			}
		}
	}
}

func (s *auditLogReaderImpl) shouldSendEvent(event *auditEvent) bool {
	return event.ObjectRef.Resource == "secrets" || event.ObjectRef.Resource == "configmaps"
}

func (s *auditLogReaderImpl) cleanupTailOnStop(tailer *tail.Tail) {
	if err := tailer.Stop(); err != nil {
		log.Errorf("Audit log tailer stopped with an error %v", err)
	}
	tailer.Cleanup()
}
