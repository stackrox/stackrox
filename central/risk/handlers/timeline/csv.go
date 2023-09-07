package timeline

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	podUtils "github.com/stackrox/rox/pkg/pods/utils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	headersAndValueGetters = []struct {
		header string
		getter func(*eventRow) string
	}{
		{
			header: "Event Timestamp",
			getter: func(r *eventRow) string { return r.eventTimestamp },
		},
		{
			header: "Event Type",
			getter: func(r *eventRow) string { return r.eventType },
		},
		{
			header: "Event Name",
			getter: func(r *eventRow) string { return r.eventName },
		},
		{
			header: "Process Args",
			getter: func(r *eventRow) string { return r.processArgs },
		},
		{
			header: "Process Parent Name",
			getter: func(r *eventRow) string { return r.processParentName },
		},
		{
			header: "Process Baselined",
			getter: func(r *eventRow) string { return r.processInBaseline },
		},
		{
			header: "Process UID",
			getter: func(r *eventRow) string { return r.processUID },
		},
		{
			header: "Process Parent UID",
			getter: func(r *eventRow) string { return r.processParentUID },
		},
		{
			header: "Container Exit Code",
			getter: func(r *eventRow) string { return r.containerExitCode },
		},
		{
			header: "Container Exit Reason",
			getter: func(r *eventRow) string { return r.containerExitReason },
		},
		{
			header: "Container ID",
			getter: func(r *eventRow) string { return r.containerID },
		},
		{
			header: "Container Name",
			getter: func(r *eventRow) string { return r.containerName },
		},
		{
			header: "Container Start Time",
			getter: func(r *eventRow) string { return r.containerStartTime },
		},
		{
			header: "Deployment ID",
			getter: func(r *eventRow) string { return r.deploymentID },
		},
		{
			header: "Pod ID",
			getter: func(r *eventRow) string { return r.podID },
		},
		{
			header: "Pod Name",
			getter: func(r *eventRow) string { return r.podName },
		},
		{
			header: "Pod Start Time",
			getter: func(r *eventRow) string { return r.podStartTime },
		},
		{
			header: "Pod Container Count",
			getter: func(r *eventRow) string { return r.podContainerCount },
		},
	}
)

// Ordering here does not matter, as we fill the columns based on above headers
// and getters
type eventRow struct {
	eventType           string
	eventName           string
	eventTimestamp      string
	processArgs         string
	processUID          string
	processParentUID    string
	processParentName   string
	processInBaseline   string
	containerExitCode   string
	containerExitReason string
	containerID         string
	containerName       string
	containerStartTime  string
	deploymentID        string
	podID               string
	podName             string
	podStartTime        string
	podContainerCount   string
}

type csvResults struct {
	*csv.GenericWriter
}

func newCSVResults(header []string) csvResults {
	return csvResults{
		GenericWriter: csv.NewGenericWriter(header, false),
	}
}

func (c *csvResults) addRow(row *eventRow) {
	value := make([]string, len(headersAndValueGetters))
	for i, e := range headersAndValueGetters {
		value[i] = e.getter(row)
	}
	c.AddValue(value)
}

// CSVHandler is an HTTP handler that outputs CSV exports of deployment timeline data
func CSVHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := loaders.WithLoaderContext(r.Context())

		rawQuery := r.URL.Query().Get("query")

		resolver := resolvers.New()
		podResolvers, err := resolver.Pods(ctx, resolvers.PaginatedQuery{Query: &rawQuery})
		if err != nil {
			csv.WriteError(w, errox.ServerError.CausedBy(err))
			return
		}
		containerResolvers, err := resolver.GroupedContainerInstances(ctx, resolvers.RawQuery{Query: &rawQuery})
		if err != nil {
			csv.WriteError(w, errox.ServerError.CausedBy(err))
			return
		}

		type podInfo struct {
			deploymentID   string
			startTime      string
			containerCount string
		}

		pods := make(map[string]*podInfo, len(podResolvers))
		for _, podResolver := range podResolvers {
			info := &podInfo{
				deploymentID:   podResolver.DeploymentId(ctx),
				containerCount: strconv.Itoa(int(podResolver.ContainerCount())),
			}
			pods[string(podResolver.Id(ctx))] = info

			started, err := podResolver.Started(ctx)
			if err != nil {
				log.Errorf("CSV will not include Pod Start Time: %v", err)
				continue
			}
			info.startTime = csv.FromGraphQLTime(started)
		}

		headers := make([]string, len(headersAndValueGetters))
		for i, e := range headersAndValueGetters {
			headers[i] = e.header
		}
		var dataRows []eventRow
		for _, containerResolver := range containerResolvers {
			containerID := string(containerResolver.ID())
			containerName := containerResolver.Name()
			containerStartTime := csv.FromGraphQLTime(containerResolver.StartTime())

			var podName, podUID, deploymentID, podStartTime, podContainerCount string

			if podID, err := podUtils.ParsePodID(containerResolver.PodID()); err != nil {
				log.Errorf("Unable to generate full CSV row for container %s: %v", containerName, utils.ShouldErr(err))
			} else {
				podName = podID.Name
				podUID = string(podID.UID)
				info := pods[podUID]
				deploymentID = info.deploymentID
				podStartTime = info.startTime
				podContainerCount = info.containerCount
			}

			for _, event := range containerResolver.Events() {
				var dataRow eventRow

				// Common fields in all types of events
				dataRow.eventName = event.Name()
				dataRow.eventTimestamp = csv.FromGraphQLTime(event.Timestamp())

				dataRow.containerID = containerID
				dataRow.containerName = containerName
				dataRow.containerStartTime = containerStartTime
				dataRow.deploymentID = deploymentID
				dataRow.podID = podUID
				dataRow.podName = podName
				dataRow.podStartTime = podStartTime
				dataRow.podContainerCount = podContainerCount

				// Handle type-specific fields.

				if _, ok := event.ToContainerRestartEvent(); ok {
					dataRow.eventType = "Container Restart"
				}

				if _, ok := event.ToPolicyViolationEvent(); ok {
					dataRow.eventType = "Policy Violation"
				}

				if processEvent, ok := event.ToProcessActivityEvent(); ok {
					dataRow.eventType = "Process Activity"

					dataRow.processArgs = processEvent.Args()
					dataRow.processUID = strconv.Itoa(int(processEvent.UID()))
					dataRow.processParentUID = strconv.Itoa(int(processEvent.ParentUID()))
					dataRow.processParentName = stringutils.PointerOrDefault(processEvent.ParentName(), "")
					dataRow.processInBaseline = strconv.FormatBool(processEvent.InBaseline())
				}

				if terminationEvent, ok := event.ToContainerTerminationEvent(); ok {
					dataRow.eventType = "Container Termination"

					dataRow.containerExitCode = strconv.Itoa(int(terminationEvent.ExitCode()))
					dataRow.containerExitReason = terminationEvent.Reason()
				}

				// Add the row to all the rows
				dataRows = append(dataRows, dataRow)
			}
		}

		// We sort the events by their timestamp (latest first)
		// and by their event types, and by their event names. If
		// we exhaust these three then we just sort by process args
		sort.Slice(dataRows, func(i, j int) bool {
			// An event is "smaller" if it has a later timestamp,
			// meaning later events pop up to the front of the slice

			// Compare timestamp. Latest comes first
			if dataRows[i].eventTimestamp != dataRows[j].eventTimestamp {
				return dataRows[i].eventTimestamp > dataRows[j].eventTimestamp
			}
			// Compare type
			if dataRows[i].eventType != dataRows[j].eventType {
				return dataRows[i].eventType < dataRows[j].eventType
			}
			// Compare name
			if dataRows[i].eventName != dataRows[j].eventName {
				return dataRows[i].eventName < dataRows[j].eventName
			}
			// If we exhaust above three, just sort by process args
			return dataRows[i].processArgs < dataRows[j].processArgs
		})
		output := newCSVResults(headers)
		for i := range dataRows {
			output.addRow(&dataRows[i])
		}
		output.Write(w, "events_export")
	}
}
