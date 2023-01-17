export type FlowEntityType = 'DEPLOYMENT' | 'EXTERNAL_ENTITIES' | 'CIDR_BLOCK';

export type IndividualFlow = {
    id: string;
    type: FlowEntityType;
    entity: string;
    entityId: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: string;
    isAnomalous: boolean;
    children?: undefined;
};

export type AggregatedFlow = {
    id: string;
    type: FlowEntityType;
    entity: string;
    entityId: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: string;
    isAnomalous: boolean;
    children: IndividualFlow[];
};

export type Flow = IndividualFlow | AggregatedFlow;
