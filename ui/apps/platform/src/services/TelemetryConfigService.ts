import axios from './instance';

const url = '/v1/telemetry/config';

export type TelemetryConfig = {
    userId: string;
    endpoint: string;
    storageKeyV1: string;
};

/**
 * Fetches telemetry data for analytics.
 */
export function fetchTelemetryConfig(): Promise<TelemetryConfig> {
    return axios.get<TelemetryConfig>(url).then(({ data }) => data);
}
