import { ContainerRuntimeInfo } from './containerRuntime.proto';
import { Taint } from './taints.proto';
import { EmbeddedVulnerability } from './vulnerability.proto';

// Node represents information about a node in the cluster.
export type Node = {
    // A unique ID identifying this node.
    id: string;
    // The (host)name of the node. Might or might not be the same as ID.
    name: string;
    // Taints on the host
    taints: Taint[];
    clusterId: string;
    clusterName: string;
    labels: Record<string, string>;
    annotations: Record<string, string>;
    // When the cluster reported the node was added
    joinedAt: string; // ISO 8601 date string
    internalIpAddresses: string[];
    externalIpAddresses: string[];
    // From NodeInfo
    containerRuntimeVersion: string; // deprecated, use containerRuntime.version
    containerRuntime: ContainerRuntimeInfo;
    kernelVersion: string;
    // From NodeInfo. Operating system reported by the node (ex: linux).
    operatingSystem: string;
    // From NodeInfo. OS image reported by the node from /etc/os-release.
    osImage: string;
    kubeletVersion: string;
    kubeProxyVersion: string;
    lastUpdated: string; // ISO 8601 date string
    k8sUpdated: string; // ISO 8601 date string
    scan: NodeScan;
    components?: number; // int32
    cves?: number; // int32
    fixableCves?: number; // int32
    priority: string; // int64
    riskScore: number; // float
    topCvss?: number; // float
};

export type NodeScan = {
    scanTime: string; // ISO 8601 date string
    operatingSystem: string;
    components: EmbeddedNodeScanComponent[];
};

export type EmbeddedNodeScanComponent = {
    name: string;
    version: string;
    vulns: EmbeddedVulnerability[];
    priority: string; // int64
    topCvss?: number; // float
    riskScore: number; // float
};
