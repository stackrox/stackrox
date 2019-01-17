import axios from 'axios';
import queryString from 'qs';

const baseUrl = '/v1/benchmarks';

/**
 * @typedef {Object} Benchmark
 * @property {!string} name benchmark name
 * @property {!boolean} available indicates if benchmark if available given the current set of clusters
 */

/**
 * Fetches list of existing benchmarks.
 *
 * @returns {Promise<Benchmark[], Error>}
 */
export async function fetchBenchmarks() {
    const configsUrl = `${baseUrl}/configs`;
    return axios.get(configsUrl).then(response => response.data.benchmarks);
}

/**
 * Fetches scans metadata for the given benchmark.
 *
 * @param {!string} benchmark benchmark
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
 * @param {!Object} benchmark
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
export function updateSchedule(schedule) {
    if (!schedule || !schedule.id) throw new Error('Schedule does not have an id');
    const scheduleUrl = `${baseUrl}/schedules/${schedule.id}`;
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
    const triggerUrl = `${baseUrl}/triggers/${benchmark.benchmarkId}`;
    return axios.post(triggerUrl, {
        clusterIds: [benchmark.clusterId]
    });
}

/**
 * Fetches a map of benchmarks for each cluster
 *
 * @returns {Promise<Object, Error>} fulfilled in case of success or rejected with an error
 */
export async function fetchBenchmarksByCluster(filters) {
    const params = queryString.stringify({ ...filters }, { encode: false, arrayFormat: 'repeat' });
    const benchmarksSummaryUrl = `${baseUrl}/summary/scans?${params}`;
    return axios.get(benchmarksSummaryUrl).then(response => response.data.clusters);
}

/**
 * Fetches benchmark scan result by scanId and checkName
 *
 * @returns {Promise<Object, Error>} fulfilled in case of success or rejected with an error
 */
export async function fetchBenchmarkCheckHostResults({ scanId, checkName }) {
    const benchmarkCheckHostsResultUrl = `${baseUrl}/scans/${scanId}/${checkName}`;
    return axios.get(benchmarkCheckHostsResultUrl).then(response => response.data);
}
