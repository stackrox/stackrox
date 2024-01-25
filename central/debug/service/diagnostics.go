package service

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	concPool "github.com/sourcegraph/conc/pool"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	invalidPathElementChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

// Helper struct which holds all k8sintrospect.File gathered during either K8S diagnostic collection or
// metrics collection.
type gatherResult struct {
	files []k8sintrospect.File
}

func (s *serviceImpl) getK8sDiagnostics(ctx context.Context, zipWriter *zipWriter, opts debugDumpOptions) error {
	gatherPool := concPool.NewWithResults[[]k8sintrospect.File]().WithContext(ctx)

	clusterNameMap, err := s.getClusterNameByIDMap(ctx)
	if err != nil {
		return err
	}

	usedNames := set.NewStringSet(centralClusterPrefix)
	for _, sensorConn := range s.sensorConnMgr.GetActiveConnections() {
		clusterName, valid := getClusterNameForSensorConnection(sensorConn, usedNames, clusterNameMap, opts)
		if !valid {
			continue
		}
		gatherPool.Go(func(ctx context.Context) ([]k8sintrospect.File, error) {
			return pullK8sDiagnosticsFilesFromSensor(ctx, clusterName, sensorConn, opts.since)
		})
	}

	// Add information about clusters for which we didn't have an active sensor connection.
	if len(clusterNameMap) > 0 {
		// This is simply creating a static k8sintrospect.File, hence the context can be safely ignored.
		gatherPool.Go(func(_ context.Context) ([]k8sintrospect.File, error) {
			return addMissingClustersInfo(clusterNameMap, opts.clusters)
		})
	}

	// Pull data from the central cluster, irrespective of whether it might have been covered by the active
	// sensor connections.
	// We currently do not have a way to specify within the connection whether the sensor is co-located within the
	// Central cluster.
	if opts.withCentral {
		gatherPool.Go(func(ctx context.Context) ([]k8sintrospect.File, error) {
			return pullCentralClusterDiagnostics(ctx, opts.since)
		})
	}

	var gatherResults [][]k8sintrospect.File
	if ctxErr := concurrency.DoInWaitable(ctx, func() {
		// The error can be safely ignored, since context cancellations will be propagated via concurrency.DoInWaitable
		// and errors are contained within the gather results as files.
		gatherResults, _ = gatherPool.Wait()
	}); ctxErr != nil {
		return ctxErr
	}

	return writeGatherResultsToZIP(ctx, zipWriter, "kubernetes", gatherResults)
}

func (s *serviceImpl) pullSensorMetrics(ctx context.Context, zipWriter *zipWriter, opts debugDumpOptions) error {
	gatherPool := concPool.NewWithResults[[]k8sintrospect.File]().WithContext(ctx)

	clusterNameMap, err := s.getClusterNameByIDMap(ctx)
	if err != nil {
		return err
	}

	usedNames := set.NewStringSet(centralClusterPrefix)
	for _, sensorConn := range s.sensorConnMgr.GetActiveConnections() {
		clusterName, valid := getClusterNameForSensorConnection(sensorConn, usedNames, clusterNameMap, opts)
		if !valid {
			continue
		}
		gatherPool.Go(func(ctx context.Context) ([]k8sintrospect.File, error) {
			return pullMetricsFromSensor(ctx, clusterName, sensorConn)
		})
	}

	var gatherResults [][]k8sintrospect.File
	if ctxErr := concurrency.DoInWaitable(ctx, func() {
		// The error can be safely ignored, since context cancellations will be propagated via concurrency.DoInWaitable
		// and errors are contained within the gather results as files.
		gatherResults, _ = gatherPool.Wait()
	}); ctxErr != nil {
		return ctxErr
	}

	return writeGatherResultsToZIP(ctx, zipWriter, "sensor-metrics", gatherResults)
}

func (s *serviceImpl) getClusterNameByIDMap(ctx context.Context) (map[string]string, error) {
	// Build an ID -> Name map for clusters.
	clusters, err := s.clusters.GetClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve cluster list")
	}

	clusterNameMap := make(map[string]string, len(clusters))
	for _, cluster := range clusters {
		clusterNameMap[cluster.GetId()] = cluster.GetName()
	}
	return clusterNameMap, nil
}

func sanitizeClusterName(rawClusterName string) string {
	return invalidPathElementChars.ReplaceAllString(rawClusterName, "_")
}

func gatherResultCallback(res *gatherResult, clusterName string) func(ctx concurrency.ErrorWaitable,
	k8sInfo *central.TelemetryResponsePayload_KubernetesInfo) error {
	return func(_ concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload_KubernetesInfo) error {
		for _, k8sInfoFile := range chunk.GetFiles() {
			res.files = append(res.files, k8sintrospect.File{
				Path:     path.Join(clusterName, k8sInfoFile.GetPath()),
				Contents: k8sInfoFile.GetContents(),
			})
		}
		return nil
	}
}

func createErrorFile(clusterName string, err error) k8sintrospect.File {
	return k8sintrospect.File{
		Path:     path.Join(clusterName, "pull-error.txt"),
		Contents: []byte(err.Error()),
	}
}

func pullK8sDiagnosticsFilesFromSensor(ctx context.Context, clusterName string, sensorConn connection.SensorConnection,
	since time.Time) ([]k8sintrospect.File, error) {
	res := &gatherResult{}
	callback := gatherResultCallback(res, clusterName)

	if !sensorConn.HasCapability(centralsensor.PullTelemetryDataCap) {
		return []k8sintrospect.File{createErrorFile(clusterName,
			errors.New("sensor does not support pulling telemetry data"))}, nil
	}
	err := sensorConn.Telemetry().PullKubernetesInfo(ctx, callback, since)
	if err != nil {
		log.Warnw("Error pulling kubernetes info from sensor", logging.ClusterName(clusterName), logging.Err(err))
		return []k8sintrospect.File{
			createErrorFile(clusterName, err),
		}, err
	}
	return res.files, nil
}

func pullMetricsFromSensor(ctx context.Context, clusterName string,
	sensorConn connection.SensorConnection) ([]k8sintrospect.File, error) {
	res := &gatherResult{}
	callback := gatherResultCallback(res, clusterName)

	if !sensorConn.HasCapability(centralsensor.PullMetricsCap) {
		return []k8sintrospect.File{
			createErrorFile(clusterName, errors.New("sensor does not support pulling metrics")),
		}, nil
	}
	err := sensorConn.Telemetry().PullMetrics(ctx, callback)
	if err != nil {
		log.Warnw("Error pulling metrics from sensor", logging.ClusterName(clusterName), logging.Err(err))
		return []k8sintrospect.File{createErrorFile(clusterName, err)}, nil
	}
	return res.files, nil
}

func addMissingClustersInfo(remainingClusterNameMap map[string]string,
	filterClusters []string) ([]k8sintrospect.File, error) {
	sb := strings.Builder{}
	sb.WriteString("Data from the following clusters is unavailable:\n")
	for _, clusterName := range remainingClusterNameMap {
		if filterClusters != nil && sliceutils.Find(filterClusters, clusterName) != -1 {
			sb.WriteString(fmt.Sprintf("- %s (not requested by user)\n", clusterName))
		} else {
			sb.WriteString(fmt.Sprintf("- %s (no active connection)\n", clusterName))
		}
	}

	missingClustersFile := k8sintrospect.File{
		Path:     "missing-clusters.txt",
		Contents: []byte(sb.String()),
	}
	return []k8sintrospect.File{missingClustersFile}, nil
}

func pullCentralClusterDiagnostics(ctx context.Context, since time.Time) ([]k8sintrospect.File, error) {
	var files []k8sintrospect.File
	cb := func(_ concurrency.ErrorWaitable, file k8sintrospect.File) error {
		files = append(files, file)
		return nil
	}

	restCfg, err := k8sutil.GetK8sInClusterConfig()
	if err == nil {
		err = k8sintrospect.Collect(ctx, mainClusterConfig, restCfg, cb, since)
	}
	if err != nil {
		return []k8sintrospect.File{createErrorFile(centralClusterPrefix, err)}, nil
	}
	return files, nil
}

// getClusterCandidate returns the cluster name based off the cluster ID associated with the given sensor connection.
// Additionally, the cluster name will be filtered by the given debug dump options.
// It will return the cluster name, a sanitized version of the cluster name without invalid path characters, and
// a bool indicating whether the cluster name is valid, i.e. non-empty, or not.
func getClusterCandidate(conn connection.SensorConnection, clusterNameMap map[string]string,
	opts debugDumpOptions) (string, string, bool) {
	clusterID := conn.ClusterID()
	clusterName := clusterNameMap[clusterID]

	if clusterName == "" {
		clusterName = fmt.Sprintf("_%s", clusterID)
	}
	delete(clusterNameMap, clusterID)

	// if there are no cluster filters, all clusters must be considered.
	if opts.clusters != nil && sliceutils.Find(opts.clusters, clusterName) == -1 {
		return "", "", false
	}

	// Make sure we use a name that doesn't clash with any other cluster name.
	return clusterName, sanitizeClusterName(clusterName), true
}

// getClusterNameForSensorConnection returns the cluster name associated with the given sensor connection.
// In case the cluster name has already been used beforehand, the name will be suffixed with an index.
// In case the cluster isn't seen as valid (i.e. it has been filtered out by the given debug dump options),
// it will return false.
func getClusterNameForSensorConnection(conn connection.SensorConnection, usedClusterNames set.Set[string],
	clusterIDToName map[string]string, opts debugDumpOptions) (string, bool) {
	_, candidateName, valid := getClusterCandidate(conn, clusterIDToName, opts)
	if !valid {
		return "", false
	}

	i := 0
	for !usedClusterNames.Add(candidateName) {
		candidateName = fmt.Sprintf("%s_%d", candidateName, i)
		i++
	}
	return candidateName, true
}

// writeGatherResultsToZIP writes the given gather results to the ZIP with the defined prefix.
// This respects context cancellation during writing the ZIP.
func writeGatherResultsToZIP(ctx context.Context, zipWriter *zipWriter, zipPrefix string,
	gatherResults [][]k8sintrospect.File) error {
	for _, files := range gatherResults {
		for _, file := range files {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				err := zipWriter.writePrefixedFileToZip(zipPrefix, file)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
