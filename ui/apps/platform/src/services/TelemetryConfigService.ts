import { TelemetryConfig } from 'types/telemetryConfigService.proto';

import axios from './instance';

const url = '/v1/telemetry/config';

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
