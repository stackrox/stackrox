export interface FlowBase {
    id: string;
    type: 'Deployment' | 'External';
    entity: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: string;
    isAnomalous: boolean;
}

export interface Flow extends FlowBase {
    children: FlowBase[];
}
