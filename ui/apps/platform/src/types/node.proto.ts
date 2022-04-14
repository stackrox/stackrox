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
    // TODO(ROX-6895): "Label" search term is ambiguous.
    labels: Record<string, string>;
    annotations: Record<string, string>;
    // When the cluster reported the node was added
    joinedAt: string; // ISO 8601 date string
    // node internal IP addresses
    internalIpAddresses: string[];
    // node external IP addresses
    external_ip_addresses: string[];
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
    // Time we received an update from Kubernetes.
    k8sUpdated: string; // ISO 8601 date string

    scan: NodeScan;
    // oneof set_components {
    components: number; // int32
    // }
    // oneof set_cves {
    cves: number; // int32
    // }
    // oneof set_fixable {
    fixableCves: number; // int32
    // }
    priority: number; // int64
    riskScore: number; // float
    // oneof set_top_cvss {
    topCvss: number; // float
    // }
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
    priority: number; // int64
    // oneof set_top_cvss {
    topCvss: number; // float
    // }
    riskScore: number; // float
};

export type NodeComponentEdge = {
    // base 64 encoded Node:Component ids.
    id: string;
};

export type NodeCVEEdge = {
    // base 64 encoded Node:CVE ids.
    id: string;
    firstNodeOccurrence: string; // ISO 8601 date string
};
