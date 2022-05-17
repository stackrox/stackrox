import { TelemetryConfig } from '../types/telemetry.proto';

import axios from './instance';

const baseURL = '/v1/telemetry';

/**
 * Get telemetry setting for product use.
 *
 * TODO return Promise<TelemetryConfiguration> when System Configuration page calls directly instead of indirectly via saga.
 */
export function fetchTelemetryConfig(): Promise<{ response: TelemetryConfig }> {
    return axios.get<TelemetryConfig>(`${baseURL}/configure`).then(({ data }) => ({
        response: data,
    }));
}

export function saveTelemetryConfig(config: TelemetryConfig): Promise<TelemetryConfig> {
    return axios
        .put<TelemetryConfig>(`${baseURL}/configure`, config)
        .then((response) => response.data);
}
