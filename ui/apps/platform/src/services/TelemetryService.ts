import { TelemetryConfig } from '../types/telemetry.proto';

import axios from './instance';

const baseURL = '/v1/telemetry';

/*
 * Fetch telemetry setting for product use.
 */
export function fetchTelemetryConfig(): Promise<TelemetryConfig> {
    return axios.get<TelemetryConfig>(`${baseURL}/configure`).then((response) => response.data);
}

/*
 * Save telemetry setting for product use.
 */
export function saveTelemetryConfig(config: TelemetryConfig): Promise<TelemetryConfig> {
    return axios
        .put<TelemetryConfig>(`${baseURL}/configure`, config)
        .then((response) => response.data);
}
