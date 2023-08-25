import { L4Protocol } from 'types/networkFlow.proto';

export const l4ProtocolLabels: Record<L4Protocol, string> = {
    L4_PROTOCOL_UNKNOWN: 'Unknown',
    L4_PROTOCOL_TCP: 'TCP',
    L4_PROTOCOL_UDP: 'UDP',
    L4_PROTOCOL_ICMP: 'ICMP',
    L4_PROTOCOL_RAW: 'Raw',
    L4_PROTOCOL_SCTP: 'SCTP',
    L4_PROTOCOL_ANY: 'Any Protocol',
};
