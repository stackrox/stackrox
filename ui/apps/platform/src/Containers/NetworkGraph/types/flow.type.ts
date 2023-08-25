import { L4Protocol } from 'types/networkFlow.proto';

export type EntityType = 'DEPLOYMENT' | 'INTERNET' | 'EXTERNAL_SOURCE';

export type FlowEntityType = 'DEPLOYMENT' | 'EXTERNAL_ENTITIES' | 'CIDR_BLOCK';

export type BaselineStatusType = 'ANOMALOUS' | 'BASELINE';

export type BaselineSimulationDiffState = 'ADDED' | 'REMOVED' | 'UNCHANGED';

export type IndividualFlow = {
    id: string;
    type: FlowEntityType;
    entity: string;
    entityId: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: L4Protocol;
    isAnomalous: boolean;
    children?: undefined;
    baselineSimulationDiffState?: BaselineSimulationDiffState;
};

export type AggregatedFlow = {
    id: string;
    type: FlowEntityType;
    entity: string;
    entityId: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: L4Protocol;
    isAnomalous: boolean;
    children: IndividualFlow[];
    baselineSimulationDiffState?: BaselineSimulationDiffState;
};

export type Entity = {
    id: string;
    type: EntityType;
    name: string;
    namespace: string;
};

export type Peer = {
    entity: Entity;
    port: number;
    protocol: L4Protocol;
    ingress: boolean;
};

export type BaselineStatus = {
    peer: Peer;
    status: BaselineStatusType;
};

export type Flow = IndividualFlow | AggregatedFlow;
