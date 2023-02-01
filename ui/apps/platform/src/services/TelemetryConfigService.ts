import axios from './instance';

const url = '/v1/telemetry/config';

type TelemetryConfig = {
    userId: string;
    endpoint: string;
    storageKeyV1: string;
};

/**
 * Fetches telemetry data for analytics.
 */
export function fetchTelemetryConfig(): Promise<{
    response: { telemetryConfig: TelemetryConfig };
}> {
    return axios.get<{ telemetryConfig: TelemetryConfig }>(url).then((response) => ({
        response: response.data,
    }));
}
