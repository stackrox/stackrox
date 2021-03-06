import { resourceTypes } from 'constants/entityTypes';
import {
    PROTOCOLS,
    networkConnections,
    networkFlowStatus,
    nodeTypes,
} from 'constants/networkGraph';

export const networkProtocolLabels = {
    [PROTOCOLS.L4_PROTOCOL_TCP]: 'TCP',
    [PROTOCOLS.L4_PROTOCOL_UDP]: 'UDP',
    [PROTOCOLS.L4_PROTOCOL_ANY]: 'Any Protocol',
};

export const networkEntityLabels = {
    [resourceTypes.DEPLOYMENT]: 'Deployment',
    [nodeTypes.EXTERNAL_ENTITIES]: 'External',
    [nodeTypes.CIDR_BLOCK]: 'External',
};

export const networkConnectionLabels = {
    [networkConnections.ACTIVE]: 'Active',
    [networkConnections.ALLOWED]: 'Allowed',
    [networkConnections.ACTIVE_AND_ALLOWED]: 'Active/Allowed',
};

export const networkFlowStatusLabels = {
    [networkFlowStatus.ANOMALOUS]: 'Anomalous',
    [networkFlowStatus.BASELINE]: 'Baseline',
    [networkFlowStatus.BLOCKED]: 'Blocked',
};
