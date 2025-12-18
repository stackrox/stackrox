import axios from './instance';

/**
 * Fetch Prometheus metrics from the /metrics endpoint
 */
export async function fetchPrometheusMetrics(): Promise<string> {
    const response = await axios.get<string>('/metrics', {
        headers: {
            Accept: 'text/plain',
        },
    });
    return response.data;
}
