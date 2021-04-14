import { Entity, Protocol, ConnectionState } from 'Containers/Network/networkTypes';

export type SimulatedBaselineStatus = 'ADDED' | 'REMOVED' | 'MODIFIED' | 'UNMODIFIED';

export type Properties = {
    port: string;
    protocol: Protocol;
    ingress: boolean;
};

export type AddedBaseline = {
    peer: {
        entity: Entity;
        added: Properties;
        state: ConnectionState;
    };
    simulatedStatus: 'ADDED';
};

export type RemovedBaseline = {
    peer: {
        entity: Entity;
        removed: Properties;
        state: ConnectionState;
    };
    simulatedStatus: 'REMOVED';
};

export type ModifiedBaseline = {
    peer: {
        entity: Entity;
        modified: {
            added: Properties;
            removed: Properties;
        };
        state: ConnectionState;
    };
    simulatedStatus: 'MODIFIED';
};

export type UnmodifiedBaseline = {
    peer: {
        entity: Entity;
        unmodified: Properties;
        state: ConnectionState;
    };
    simulatedStatus: 'UNMODIFIED';
};

export type SimulatedBaseline =
    | AddedBaseline
    | RemovedBaseline
    | ModifiedBaseline
    | UnmodifiedBaseline;
