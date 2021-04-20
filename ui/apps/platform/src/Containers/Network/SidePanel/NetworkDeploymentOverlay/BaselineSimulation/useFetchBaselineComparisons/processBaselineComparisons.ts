import { FilterState } from 'Containers/Network/networkTypes';
import { filterLabels } from 'constants/networkFilterModes';
import {
    SimulatedBaseline,
    SimulatedBaselineStatus,
    Properties,
} from '../SimulatedNetworkBaselines/baselineSimulationTypes';

type DeploymentEntity = {
    id: string;
    type: 'DEPLOYMENT';
    deployment: {
        name: string;
        namespace: string;
    };
};

type ExternalSourceEntity = {
    id: string;
    type: 'EXTERNAL_SOURCE';
    externalSource: {
        name: string;
        cidr: string;
    };
};

type InternetEntity = {
    id: string;
    type: 'INTERNET';
};

type AddedRemovedBaselineResponse = {
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity;
    properties: [Properties];
};

export type ReconciledBaselineResponse = {
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity;
    added: [Properties];
    removed: [Properties];
    unchanged: [Properties];
};

export type BaselineResponse = {
    added: AddedRemovedBaselineResponse[];
    removed: AddedRemovedBaselineResponse[];
    reconciled: ReconciledBaselineResponse[];
};

function getEntityNameByType(
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity
): string {
    if (entity.type === 'DEPLOYMENT') {
        return entity.deployment.name;
    }
    if (entity.type === 'EXTERNAL_SOURCE') {
        return entity.externalSource.cidr;
    }
    if (entity.type === 'INTERNET') {
        return 'External Sources';
    }
    throw new Error('Could not get name of entity');
}

function getEntityNamespaceByType(
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity
): string {
    if (entity.type === 'DEPLOYMENT') {
        return entity.deployment.namespace;
    }
    if (entity.type === 'EXTERNAL_SOURCE' || entity.type === 'INTERNET') {
        return '-';
    }
    throw new Error('Could not get namespace of entity');
}

function processBaseline(
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity,
    properties: Properties[],
    filterState: FilterState,
    simulatedStatus: SimulatedBaselineStatus
): SimulatedBaseline[] {
    return properties.reduce((acc, property) => {
        acc.push({
            peer: {
                entity: {
                    id: entity.id,
                    type: entity.type,
                    name: getEntityNameByType(entity),
                    namespace: getEntityNamespaceByType(entity),
                },
                ...property,
                state: filterLabels[filterState],
            },
            simulatedStatus,
        } as SimulatedBaseline);
        return acc;
    }, [] as SimulatedBaseline[]);
}

function processAddedRemovedBaselines(
    baselines: BaselineResponse['added'] | BaselineResponse['removed'],
    filterState: FilterState,
    simulatedStatus: SimulatedBaselineStatus
): SimulatedBaseline[] {
    return baselines.reduce((acc, baseline) => {
        const { entity, properties } = baseline;
        return [...acc, ...processBaseline(entity, properties, filterState, simulatedStatus)];
    }, [] as SimulatedBaseline[]);
}

function processReconciledBaselines(
    baselines: BaselineResponse['reconciled'],
    filterState: FilterState
): SimulatedBaseline[] {
    return baselines.reduce((acc, baseline) => {
        const { entity, added, removed, unchanged } = baseline;
        const addedBaselines = processBaseline(entity, added, filterState, 'ADDED');
        const removedBaselines = processBaseline(entity, removed, filterState, 'REMOVED');
        const unmodifiedBaselines = processBaseline(entity, unchanged, filterState, 'UNMODIFIED');
        return [...acc, ...addedBaselines, ...removedBaselines, ...unmodifiedBaselines];
    }, [] as SimulatedBaseline[]);
}

function processBaselineComparisons(
    { added, removed, reconciled }: BaselineResponse,
    filterState: FilterState
): SimulatedBaseline[] {
    const result = [
        ...processAddedRemovedBaselines(added, filterState, 'ADDED'),
        ...processAddedRemovedBaselines(removed, filterState, 'REMOVED'),
        ...processReconciledBaselines(reconciled, filterState),
    ];
    return result;
}

export default processBaselineComparisons;
