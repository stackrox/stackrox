type Flows = 'anomalous' | 'baseline';

type Directionality = 'egress' | 'ingress';

type Protocols = 'TCP' | 'UDP';

type Ports = string; // number format

export type FilterValue = Flows | Directionality | Protocols | Ports;

export type AdvancedFlowsFilterType = {
    flows: Flows[];
    directionality: Directionality[];
    protocols: Protocols[];
    ports: Ports[];
};
