type Directionality = 'egress' | 'ingress';

type Flow = 'anomalous' | 'baseline';

type Protocols = 'L4_PROTOCOL_TCP' | 'L4_PROTOCOL_UDP';

type Ports = string; // number format

export type FilterValue = Directionality | Protocols | Ports;

export type AdvancedFlowsFilterType = {
    directionality: Directionality[];
    flows: Flow[];
    protocols: Protocols[];
    ports: Ports[];
};
