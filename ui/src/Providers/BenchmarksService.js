import axios from 'axios';
/**
 * Service for Benchmarks.
 */

export default function retrieveBenchmarks() {
    const baseUrl = '/v1';
    const clustersUrl = `${baseUrl}/clusters`;
    const configsUrl = `${baseUrl}/benchmarks/configs`;

    return Promise.all([axios.get(clustersUrl), axios.get(configsUrl)]).then(
        ([clusterResponse, configResponse]) => {
            const { clusters } = clusterResponse.data;
            const clusterTypes = new Set(clusters.map(c => c.type));
            const { benchmarks } = configResponse.data;
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
    );
}
