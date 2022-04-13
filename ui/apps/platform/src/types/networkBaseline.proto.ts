import { L4Protocol, NetworkEntity } from './networkFlow.proto';

export type NetworkBaselineConnectionProperties = {
    // Whether this connection is an ingress/egress, from the PoV of the deployment whose baseline this is in
    ingress: boolean;

    // May be 0 if not applicable (e.g., icmp), and denotes the destination port
    port: number; // uint32
    protocol: L4Protocol;
};

export type NetworkBaselinePeer = {
    entity: NetworkEntity;

    // Will always have at least one element
    properties: NetworkBaselineConnectionProperties[];
};

// NetworkBaseline represents a network baseline of a deployment. It contains all
// the baseline peers and their respective connections.
// next available tag: 8
export type NetworkBaseline = {
    // This is the ID of the baseline.
    deploymentId: string;

    clusterId: string;
    namespace: string;

    peers: NetworkBaselinePeer[];

    // A list of peers that will never be added to the baseline.
    // For now, this contains peers that the user has manually removed.
    // This is used to ensure we don't add it back in the event we see the flow again.
    forbiddenPeers: NetworkBaselinePeer[];

    observationPeriodEnd: string; // ISO 8601 date string

    // Indicates if this baseline has been locked by user.
    // Here locking means:
    // 1: Do not let system automatically add any allowed peer to baseline
    // 2: Start reporting violations on flows that are not in the baseline
    locked: boolean;

    deploymentName: string;
};
