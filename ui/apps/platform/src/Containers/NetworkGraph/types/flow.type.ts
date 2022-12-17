export type IndividualFlow = {
    id: string;
    type: 'Deployment' | 'External';
    entity: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: string;
    isAnomalous: boolean;
    children?: undefined;
};

export type AggregatedFlow = {
    id: string;
    type: 'Deployment' | 'External';
    entity: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: string;
    isAnomalous: boolean;
    children: IndividualFlow[];
};

export type Flow = IndividualFlow | AggregatedFlow;
