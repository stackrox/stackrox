import { LabelSelector, MatchLabelsSelector } from './labels.proto';

export type NetworkPolicy = {
    id: string;
    name: string;
    clusterId: string;
    clusterName: string;
    namespace: string;
    labels: Record<string, string>;
    annotations: Record<string, string>;
    spec: NetworkPolicySpec;
    yaml: string;
    apiVersion: string;
    created: string; // ex: 2022-11-14T18:01:26Z
};

export type NetworkPolicySpec = {
    podSelector: LabelSelector;
    ingress: NetworkPolicyIngressRule[];
    egress: NetworkPolicyEgressRule[];
    policyTypes: NetworkPolicyType[];
};

export type NetworkPolicyIngressRule = {
    ports: NetworkPolicyPort[];
    from: NetworkPolicyPeer[];
};

export type NetworkPolicyEgressRule = {
    ports: NetworkPolicyPort[];
    to: NetworkPolicyPeer[];
};

export type NetworkPolicyType =
    | 'UNSET_NETWORK_POLICY_TYPE'
    | 'INGRESS_NETWORK_POLICY_TYPE'
    | 'EGRESS_NETWORK_POLICY_TYPE';

export type NetworkPolicyPort = {
    protocol: Protocol;
    portRef: {
        port?: number;
        portName?: string;
    };
};

export type NetworkPolicyPeer = {
    podSelector: LabelSelector | MatchLabelsSelector;
    namespace_selector: LabelSelector;
    ipBlock: IPBlock;
};

export type Protocol = 'UNSET_PROTOCOL' | 'TCP_PROTOCOL' | 'UDP_PROTOCOL' | 'SCTP_PROTOCOL';

export type IPBlock = {
    cidr: string;
    except: string[];
};

export type NetworkPolicyReference = {
    namespace: string;
    name: string;
};

export type NetworkPolicyModification = {
    applyYaml: string;
    toDelete: NetworkPolicyReference[];
};
