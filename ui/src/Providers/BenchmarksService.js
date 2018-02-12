import axios from 'axios';

import fetchClusters from 'Providers/ClustersService';

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
        fetchClusters(),
        axios.get(configsUrl).then(response => response.data.benchmarks)
    ]);
    const clusterTypes = new Set(clusters.map(c => c.type));
    return benchmarks.map(benchmark => {
        const available = benchmark.clusterTypes.reduce(
            (val, type) => val || clusterTypes.has(type),
            false
        );
        return {
            name: benchmark.name,
            available
        };
    });
}

/**
 * Fetches scans metadata for the given benchmark.
 *
 * @param {!string} benchmarkName name of the benchmark
 * @returns {Promise<Object, Error>} fulfilled with scan metadata (type defined in .proto)
 */
export function fetchScanMetadata(benchmarkName) {
    const scanMetadataUrl = `${baseUrl}/scans?benchmark=${benchmarkName}`;
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
 * @param {!string} benchmarkName name of the benchmark
 * @returns {Promise<?ScanWithMetadata, Error>} fulfilled with scan data and metadata or `null` if no scans exist
 */
export async function fetchLastScan(benchmarkName) {
    const scanMetadata = await fetchScanMetadata(benchmarkName);
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
 * @param {!string} benchmarkName name of the benchmark
 * @returns {Promise<?Object, Error>} fulfilled with schedule data or `null` if schedule isn't configured
 */
export function fetchSchedule(benchmarkName) {
    const scheduleUrl = `${baseUrl}/schedules/${benchmarkName}`;
    return axios
        .get(scheduleUrl)
        .then(response => response.data)
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
    return axios.post(schedulesUrl, schedule);
}

/**
 * Updates scan schedule for the given benchmark.
 *
 * @param {!string} benchmarkName name of the benchmark with which the schedule is associated
 * @param {!Object} schedule schedule (as defined in .proto)
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function updateSchedule(benchmarkName, schedule) {
    const scheduleUrl = `${baseUrl}/schedules/${benchmarkName}`;
    return axios.put(scheduleUrl, schedule);
}

/**
 * Deletes any scan schedule associated with the given benchmark.
 *
 * @param {!string} benchmarkName name of the benchmark with which the schedule is associated
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function deleteSchedule(benchmarkName) {
    const scheduleUrl = `${baseUrl}/schedules/${benchmarkName}`;
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
 * @param {!string} benchmarkName name of the benchmark with which the schedule is associated
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success of trigger (not scanning) operation
 * or rejected with an error
 */
export function triggerScan(benchmarkName) {
    const triggerUrl = `${baseUrl}/triggers/${benchmarkName}`;
    return axios.post(triggerUrl, {});
}
