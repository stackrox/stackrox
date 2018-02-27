export const alerts = {
    countsByCluster: /v1\/alerts\/summary\/counts\?group_by=CLUSTER.*/,
    countsByCategory: /v1\/alerts\/summary\/counts\?group_by=CATEGORY.*/
};

export const clusters = {
    list: 'v1/clusters'
};

export const benchmarks = {
    configs: 'v1/benchmarks/configs',
    scans: 'v1/benchmarks/scans/*',
    cisDockerScans: /v1\/benchmarks\/scans\?benchmark=CIS Docker v1\.1\.0 Benchmark/,
    triggers: 'v1/benchmarks/triggers/*'
};
