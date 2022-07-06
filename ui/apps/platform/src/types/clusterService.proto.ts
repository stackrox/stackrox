import { Cluster } from './cluster.proto';

export type DeploymentFormat = 'KUBECTL' | 'HELM' | 'HELM_VALUES';

export type LoadBalancerType = 'NONE' | 'LOAD_BALANCER' | 'NODE_PORT' | 'ROUTE';

export type DecommissionedClusterRetentionInfo =
    | {
          // Cluster will not be deleted even if sensor status remains UNHEALTHY:
          // because it has an ignore label, if true
          // because system configuration is never delete, if false
          isExcluded: boolean;
      }
    | {
          // Cluster will be deleted if sensor status remains UNHEALTHY for the number of days.
          daysUntilDeletion: number; // int32
      }
    | null; // Cluster does not have sensor status UNHEALTHY.

export type ClusterResponse = {
    cluster: Cluster;
    clusterRetentionInfo: DecommissionedClusterRetentionInfo;
};

export type ClusterIdToRetentionInfo = Record<string, DecommissionedClusterRetentionInfo>;

export type ClustersResponse = {
    clusters: Cluster[];
    // Map secured clusters whose sensors have 'UNHEALTHY' status by clusterId to retention info.
    clusterIdToRetentionInfo: ClusterIdToRetentionInfo;
};

export type ClusterDefaultsResponse = {
    mainImageRepository: string;
    collectorImageRepository: string;
    kernelSupportAvailable: boolean;
};
