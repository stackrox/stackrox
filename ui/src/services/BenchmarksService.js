import axios from 'axios';

import { fetchClusters } from 'services/ClustersService';

const baseUrl = '/v1/benchmarks';

/**
 * @typedef {Object} Benchmark
 * @property {!string} name benchmark name
 * @property {!boolean} available indicates if benchmark if available given the current set of clusters
 */

/**
 * Fetches list of existing benchmarks with their availablity.
 *
 * @returns {Promise<Benchmark[], Error>}
 */
export async function fetchBenchmarks() {
    const configsUrl = `${baseUrl}/configs`;

    const [clusters, benchmarks] = await Promise.all([
        fetchClusters().then(({ response }) => response.clusters),
        axios.get(configsUrl).then(response => response.data.benchmarks)
    ]);
    const clusterTypes = new Set(clusters.map(c => c.type));
    return {
        response: benchmarks.map(benchmark => {
            const available = benchmark.clusterTypes.reduce(
                (val, type) => val || clusterTypes.has(type),
                false
            );
            return {
                id: benchmark.id,
                name: benchmark.name,
                available
            };
        })
    };
}

/**
 * Fetches scans metadata for the given benchmark.
 *
 * @param {!string} benchmarkId id of the benchmark
 * @returns {Promise<Object, Error>} fulfilled with scan metadata (type defined in .proto)
 */
export function fetchScanMetadata(benchmark) {
    const scanMetadataUrl = `${baseUrl}/scans?benchmarkId=${benchmark.benchmarkId}${
        benchmark.clusterId ? `&clusterIds=${benchmark.clusterId}` : ''
    }`;
    return axios.get(scanMetadataUrl).then(response => response.data.scanMetadata);
}

/**
 * @typedef {Object} ScanWithMetadata
 * @property {!Object} data benchmark checks for this scan (type defined in .proto)
 * @property {!Object} metadata scan metdata (type defined in .proto)
 */

/**
 * Fetches scans metadata for the given benchmark.
 *
 * @param {!string} benchmarkId id of the benchmark
 * @returns {Promise<?ScanWithMetadata, Error>} fulfilled with scan data and metadata or `null` if no scans exist
 */
export async function fetchLastScan(benchmarkId) {
    const scanMetadata = await fetchScanMetadata(benchmarkId);
    if (!scanMetadata || !scanMetadata.length) return null;
    const lastScanMetadata = scanMetadata[0];
    const scanUrl = `${baseUrl}/scans/${lastScanMetadata.scanId}`;
    const lastScanData = await axios.get(scanUrl).then(response => response.data);
    return {
        metadata: lastScanMetadata,
        data: lastScanData
    };
}

/**
 * Fetches scan schedule for the given benchmark.
 *
 * @param {!string} benchmarkId id of the benchmark
 * @returns {Promise<?Object, Error>} fulfilled with schedule data or `null` if schedule isn't configured
 */
export async function fetchSchedule(benchmark) {
    const scheduleUrl = `${baseUrl}/schedules?benchmarkId=${benchmark.benchmarkId}${
        benchmark.clusterId ? `&clusterIds=${benchmark.clusterId}` : ''
    }`;
    const value = await axios
        .get(scheduleUrl)
        .then(response => response.data)
        .catch(error => {
            if (error.response && error.response.status === 404) {
                return null; // schedule doesn't exist
            }
            return Promise.reject(error);
        });
    return value;
}

/**
 * Fetches scan schedules for the given benchmark.
 *
 * @param {!string} benchmarkId id of the benchmark
 * @returns {Promise<?Object, Error>} fulfilled with schedule data or `null` if schedule isn't configured
 */
export function fetchSchedules(benchmark) {
    const scheduleUrl = `${baseUrl}/schedules?benchmarkIds=${benchmark.benchmarkId}${
        benchmark.clusterId ? `&clusterIds=${benchmark.clusterId}` : ''
    }`;
    return axios
        .get(scheduleUrl)
        .then(response => response.data.schedules)
        .catch(error => {
            if (error.response && error.response.status === 404) {
                return null; // schedule doesn't exist
            }
            return Promise.reject(error);
        });
}

/**
 * Creates new scan schedule.
 *
 * @param {!Object} schedule schedule (as defined in .proto)
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function createSchedule(schedule) {
    const schedulesUrl = `${baseUrl}/schedules`;
    const formattedSchedule = Object.assign(schedule, {
        benchmark_id: schedule.benchmarkId,
        benchmark_name: schedule.benchmarkName
    });
    return axios.post(schedulesUrl, formattedSchedule);
}

/**
 * Updates scan schedule for the given benchmark.
 *
 * @param {!string} benchmarkId id of the benchmark with which the schedule is associated
 * @param {!Object} schedule schedule (as defined in .proto)
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function updateSchedule(benchmarkId, schedule) {
    const scheduleUrl = `${baseUrl}/schedules/${benchmarkId}`;
    const formattedSchedule = Object.assign(schedule, {
        benchmark_id: schedule.benchmarkId,
        benchmark_name: schedule.benchmarkName
    });
    return axios.put(scheduleUrl, formattedSchedule);
}

/**
 * Deletes any scan schedule associated with the given benchmark.
 *
 * @param {!string} benchmarkId id of the benchmark with which the schedule is associated
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function deleteSchedule(benchmarkId) {
    const scheduleUrl = `${baseUrl}/schedules/${benchmarkId}`;
    return axios.delete(scheduleUrl).catch(error => {
        if (error.response && error.response.status === 404) {
            return null; // schedule didn't exist, no harm done
        }
        return Promise.reject(error);
    });
}

/**
 * Triggers benchmark scanning.
 *
 * @param {!string} benchmarkId id of the benchmark with which the schedule is associated
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success of trigger (not scanning) operation
 * or rejected with an error
 */
export function triggerScan(benchmark) {
    const triggerUrl = `${baseUrl}/triggers/${benchmark.benchmarkId}${
        benchmark.clusterId ? `?clusterIds=${benchmark.clusterId}` : ''
    }`;
    return axios.post(triggerUrl, {});
}

/**
 * Fetches a map of benchmarks and last scans for them
 *
 * @returns {Promise<Object, Error>} fulfilled in case of success or rejected with an error
 */
export async function fetchLastScansByBenchmark() {
    const allBenchmarks = await fetchBenchmarks();
    const lastScans = await Promise.all(
        allBenchmarks.response.filter(b => b.available).map(b => {
            const promise = fetchLastScan({ benchmarkId: b.id });
            promise.then(obj => {
                if (obj) return Object.assign(obj, { benchmarkName: b.name });
                return obj;
            });
            return promise;
        })
    );
    const benchmarks = lastScans.reduce(
        (result, scan) => (scan ? { ...result, [scan.benchmarkName]: [scan.data] } : result),
        {}
    );
    return { response: benchmarks };
}
