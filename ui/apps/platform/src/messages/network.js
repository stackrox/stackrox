import { resourceTypes } from 'constants/entityTypes';
import { PROTOCOLS } from 'constants/networkGraph';

export const networkProtocolLabels = {
    [PROTOCOLS.L4_PROTOCOL_TCP]: 'TCP',
    [PROTOCOLS.L4_PROTOCOL_UDP]: 'UDP',
    [PROTOCOLS.L4_PROTOCOL_ANY]: 'Any Protocol',
};

export const networkEntities = {
    [resourceTypes.DEPLOYMENT]: 'Deployment',
};
