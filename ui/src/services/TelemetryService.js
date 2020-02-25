import axios from './instance';

const baseURL = '/v1/telemetry';

/**
 * Downloads diagnostic zip.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export function fetchTelemetryConfig() {
    return axios.get(`${baseURL}/configure`).then(({ data }) => ({
        response: data
    }));
}

export function saveTelemetryConfig(config) {
    return axios.put(`${baseURL}/configure`, config);
}
