package auditlog

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/nxadm/tail"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()

	// stagesAllowList is the set of stages that will be sent.
	// Currently only `ResponseComplete` ("The response body has been completed, and no more bytes will be sent.") OR
	// `Panic` ("Events generated when a panic occurred.") are supported
	stagesAllowList = set.NewFrozenStringSet("ResponseComplete", "Panic")

	// resourceTypesAllowList is set of resources that will be sent.
	resourceTypesAllowList = set.NewFrozenStringSet("secrets", "configmaps", "clusterrolebindings", "clusterroles", "networkpolicies", "securitycontextconstraints", "egressfirewalls")

	// verbsDenyList is the set of verbs that will NOT be sent if encountered.
	verbsDenyList = set.NewFrozenStringSet("WATCH", "LIST")

	// verbsDenyListWithGet is the set of verbs that will NOT be sent if encountered.
	verbsDenyListWithGet = set.NewFrozenStringSet("WATCH", "LIST", "GET")

	// verbsDenyListPerResource is the set of verbs that will NOT be sent if encountered.
	verbsDenyListPerResource = map[string]set.FrozenStringSet{
		"secrets":                    verbsDenyList,
		"configmaps":                 verbsDenyList,
		"clusterrolebindings":        verbsDenyListWithGet,
		"clusterroles":               verbsDenyListWithGet,
		"networkpolicies":            verbsDenyListWithGet,
		"securitycontextconstraints": verbsDenyListWithGet,
		"egressfirewalls":            verbsDenyListWithGet,
	}
)

type auditLogReaderImpl struct {
	logPath    string
	stopper    concurrency.Stopper
	sender     auditLogSender
	startState *storage.AuditLogFileState
}

func (s *auditLogReaderImpl) StartReader(ctx context.Context) (bool, error) {
	t, err := tail.TailFile(s.logPath, tail.Config{
		ReOpen:    true,
		MustExist: true,
		Follow:    true,
	})

	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Audit log file %s doesn't exist on this compliance node", s.logPath)
			err = nil
		} else {
			// handle other errors
			log.Errorf("Failed to open file: %v", err)
		}
		s.stopper.Flow().StopWithError(err)
		s.stopper.Flow().ReportStopped()
		return false, err
	}

	go s.readAndForwardAuditLogs(ctx, t)
	return true, nil
}

func (s *auditLogReaderImpl) StopReader() {
	s.stopper.Client().Stop()
	_ = s.stopper.Client().Stopped().Wait()
}

func (s *auditLogReaderImpl) readAndForwardAuditLogs(ctx context.Context, tailer *tail.Tail) {
	defer s.stopper.Flow().ReportStopped()
	defer s.cleanupTailOnStop(tailer)

	if s.startState != nil {
		log.Infof("Starting audit log reader with start time %+v and start id %s", protoutils.NewWrapper(s.startState.CollectLogsSince).String(), s.startState.LastAuditId)
	} else {
		log.Infof("Starting audit log reader with no start state")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopper.Flow().StopRequested():
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

			if strings.ToLower(auditLine.Verb) == "get" && strings.ToLower(auditLine.ObjectRef.Resource) == "configmaps" {
				log.Infof("Going to send event with id %s, for res name %s at time %+v", auditLine.AuditID, auditLine.ObjectRef.Name, protoutils.NewWrapper(eventTS).String())
			}

			if err := s.sender.Send(ctx, &auditLine); err != nil {
				// It's very likely that this failure is due to Sensor being unavailable
				// In that case when Sensor next comes available it will ask to restart from the last
				// message it got. Therefore, this event will end up being sent at that point.
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
				log.Infof("Matched up start state in reader with start time %+v and start id %s", protoutils.NewWrapper(s.startState.CollectLogsSince).String(), s.startState.LastAuditId)
				s.startState = nil
			}
			// in either case don't send
			return false
		}
		// otherwise the time is after prev state so clear out state and send (if it matches other conditions)
		log.Infof("Currently at audit log ts %+v which is after start since time %+v", protoutils.NewWrapper(eventTS), protoutils.NewWrapper(s.startState.CollectLogsSince).String())
		s.startState = nil
	}

	// Only send when both the stage and resource type are in their corresponding allow-list
	// and when verb is not disallowed
	return stagesAllowList.Contains(event.Stage) &&
		resourceTypesAllowList.Contains(event.ObjectRef.Resource) &&
		!verbsDenyListPerResource[event.ObjectRef.Resource].Contains(strings.ToUpper(event.Verb))
}

func (s *auditLogReaderImpl) cleanupTailOnStop(tailer *tail.Tail) {
	// Only want to call tailer.Stop here as that stops the tailing. However, do _not_ call tailer.Cleanup() because as the docs mention
	// "If you plan to re-read a file, don't call Cleanup in between."
	// Cleanup is recommended for after the process exits, but that's not strictly necessary as if the process exits the container will be restarted anyway
	// See https://pkg.go.dev/github.com/nxadm/tail#Tail.Cleanup for details on Cleanup()
	if err := tailer.Stop(); err != nil {
		log.Errorf("Audit log tailer stopped with an error %v", err)
	}
}
