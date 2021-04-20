import { Entity, Protocol, ConnectionState } from 'Containers/Network/networkTypes';

export type SimulatedBaselineStatus = 'ADDED' | 'REMOVED' | 'UNMODIFIED';

export type Properties = {
    port: string;
    protocol: Protocol;
    ingress: boolean;
};

export type SimulatedBaseline = {
    peer: {
        entity: Entity;
        port: string;
        protocol: Protocol;
        ingress: boolean;
        state: ConnectionState;
    };
    simulatedStatus: 'ADDED' | 'REMOVED' | 'UNMODIFIED';
};
