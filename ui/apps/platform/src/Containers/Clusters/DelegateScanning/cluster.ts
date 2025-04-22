import { DelegatedRegistryCluster } from 'services/DelegatedRegistryConfigService';

/* eslint-disable no-nested-ternary */
// Caller is responsible to handle special case of empty string.
export function getClusterName(clusters: DelegatedRegistryCluster[], clusterId: string) {
    const cluster = clusters.find((cluster) => cluster.id === clusterId);
    return cluster === undefined
        ? clusterId
        : cluster.isValid
          ? cluster.name
          : `${cluster.name} (Not available for scanning)`;
}
/* eslint-enable no-nested-ternary */
