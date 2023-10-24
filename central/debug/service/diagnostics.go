package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path"
	"regexp"
	"time"

	"github.com/pkg/errors"
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

func sanitizeClusterName(rawClusterName string) string {
	return invalidPathElementChars.ReplaceAllString(rawClusterName, "_")
}

func filePayloadCallback(filesC chan<- k8sintrospect.File, clusterName string) func(ctx concurrency.ErrorWaitable, k8sInfo *central.TelemetryResponsePayload_KubernetesInfo) error {
	return func(ctx concurrency.ErrorWaitable, k8sInfo *central.TelemetryResponsePayload_KubernetesInfo) error {
		for _, k8sInfoFile := range k8sInfo.GetFiles() {
			file := k8sintrospect.File{
				Path:     path.Join(clusterName, k8sInfoFile.GetPath()),
				Contents: k8sInfoFile.GetContents(),
			}
			select {
			case filesC <- file:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}

func pullK8sDiagnosticsFilesFromSensor(ctx context.Context, clusterName string, sensorConn connection.SensorConnection,
	filesC chan<- k8sintrospect.File, wg *concurrency.WaitGroup, since time.Time) {
	defer wg.Add(-1)

	callback := filePayloadCallback(filesC, clusterName)

	var err error
	if sensorConn.HasCapability(centralsensor.PullTelemetryDataCap) {
		err = sensorConn.Telemetry().PullKubernetesInfo(ctx, callback, since)
	} else {
		err = errors.New("sensor does not support pulling telemetry data")
	}

	if err != nil {
		log.Warnw("Error pulling kubernetes info from sensor", logging.ClusterName(clusterName), logging.Err(err))
		errFile := k8sintrospect.File{
			Path:     path.Join(clusterName, "pull-error.txt"),
			Contents: []byte(err.Error()),
		}

		select {
		case filesC <- errFile:
		case <-ctx.Done():
		}
	}
}

func pullMetricsFromSensor(ctx context.Context, clusterName string, sensorConn connection.SensorConnection,
	filesC chan<- k8sintrospect.File, wg *concurrency.WaitGroup) {
	defer wg.Add(-1)

	callback := filePayloadCallback(filesC, clusterName)

	var err error
	if sensorConn.HasCapability(centralsensor.PullMetricsCap) {
		err = sensorConn.Telemetry().PullMetrics(ctx, callback)
	} else {
		err = errors.New("sensor does not support pulling metrics")
	}

	if err != nil {
		log.Warnw("Error pulling metrics from sensor", logging.ClusterName(clusterName), logging.Err(err))
		errFile := k8sintrospect.File{
			Path:     path.Join(clusterName, "pull-error.txt"),
			Contents: []byte(err.Error()),
		}

		select {
		case filesC <- errFile:
		case <-ctx.Done():
		}
	}
}

func addMissingClustersInfo(ctx context.Context, remainingClusterNameMap map[string]string,
	filesC chan<- k8sintrospect.File, wg *concurrency.WaitGroup, filterClusters []string) {
	defer wg.Add(-1)

	var missingClustersFileContents bytes.Buffer
	fmt.Fprintln(&missingClustersFileContents, "Data from the following clusters is unavailable:")
	for _, clusterName := range remainingClusterNameMap {
		if filterClusters != nil && sliceutils.Find(filterClusters, clusterName) != -1 {
			fmt.Fprintf(&missingClustersFileContents, "- %s (not requested by user)\n", clusterName)
		} else {
			fmt.Fprintf(&missingClustersFileContents, "- %s (no active connection)\n", clusterName)
		}
	}

	missingClustersFile := k8sintrospect.File{
		Path:     "missing-clusters.txt",
		Contents: missingClustersFileContents.Bytes(),
	}

	select {
	case filesC <- missingClustersFile:
	case <-ctx.Done():
	}
}

func pullCentralClusterDiagnostics(ctx context.Context, filesC chan<- k8sintrospect.File, wg *concurrency.WaitGroup, since time.Time) {
	defer wg.Add(-1)

	restCfg, err := k8sutil.GetK8sInClusterConfig()
	if err == nil {
		err = k8sintrospect.Collect(ctx, mainClusterConfig, restCfg, k8sintrospect.SendToChan(filesC), since)
	}
	if err != nil {
		errFile := k8sintrospect.File{
			Path:     path.Join(centralClusterPrefix, "collect-error.txt"),
			Contents: []byte(err.Error()),
		}
		select {
		case filesC <- errFile:
		case <-ctx.Done():
		}
	}
}

func (s *serviceImpl) getClusterNameMap(ctx context.Context) (map[string]string, error) {
	// Build an ID -> Name map for clusters
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

func getClusterCandidate(conn connection.SensorConnection, clusterNameMap map[string]string, opts debugDumpOptions) (string, string, bool) {
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

func (s *serviceImpl) pullSensorMetrics(ctx context.Context, zipWriter *zip.Writer, opts debugDumpOptions) error {
	filesC := make(chan k8sintrospect.File)
	clusterNameMap, err := s.getClusterNameMap(ctx)
	if err != nil {
		return err
	}

	// Pull telemetry data from all active sensor connections
	var wg concurrency.WaitGroup
	usedNames := set.NewStringSet(centralClusterPrefix)

	for _, sensorConn := range s.sensorConnMgr.GetActiveConnections() {
		clusterName, candidateName, valid := getClusterCandidate(sensorConn, clusterNameMap, opts)
		if !valid {
			continue
		}

		i := 0
		for !usedNames.Add(candidateName) {
			candidateName = fmt.Sprintf("%s_%d", clusterName, i)
			i++
		}
		clusterName = candidateName

		wg.Add(1)
		go pullMetricsFromSensor(ctx, clusterName, sensorConn, filesC, &wg)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wg.Done():
			log.Info("Finished writing Sensor data to diagnostic bundle")
			return nil
		case file := <-filesC:
			err := writePrefixedFileToZip(zipWriter, "sensor-metrics", file)
			if err != nil {
				return err
			}
		}
	}
}

func (s *serviceImpl) getK8sDiagnostics(ctx context.Context, zipWriter *zip.Writer, opts debugDumpOptions) error {
	filesC := make(chan k8sintrospect.File)

	clusterNameMap, err := s.getClusterNameMap(ctx)
	if err != nil {
		return err
	}

	// Pull telemetry data from all active sensor connections
	var wg concurrency.WaitGroup
	usedNames := set.NewStringSet(centralClusterPrefix)

	for _, sensorConn := range s.sensorConnMgr.GetActiveConnections() {
		clusterName, candidateName, valid := getClusterCandidate(sensorConn, clusterNameMap, opts)
		if !valid {
			continue
		}

		i := 0
		for !usedNames.Add(candidateName) {
			candidateName = fmt.Sprintf("%s_%d", clusterName, i)
			i++
		}
		clusterName = candidateName

		wg.Add(1)
		go pullK8sDiagnosticsFilesFromSensor(ctx, clusterName, sensorConn, filesC, &wg,
			opts.since)
	}

	// Add information about clusters for which we didn't have an active sensor connection.
	if len(clusterNameMap) > 0 {
		wg.Add(1)
		go addMissingClustersInfo(ctx, clusterNameMap, filesC, &wg, opts.clusters)
	}

	// Pull data from the central cluster.
	// TODO: It would be nice if we could add a flag to a sensor connection that indicates whether this sensor is
	// running colocated with central, so we could skip this.
	if opts.withCentral {
		wg.Add(1)
		go pullCentralClusterDiagnostics(ctx, filesC, &wg, opts.since)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wg.Done():
			log.Info("Finished writing Kubernetes data to diagnostic bundle")
			return nil
		case file := <-filesC:
			err := writePrefixedFileToZip(zipWriter, "kubernetes", file)
			if err != nil {
				return err
			}
		}
	}
}
