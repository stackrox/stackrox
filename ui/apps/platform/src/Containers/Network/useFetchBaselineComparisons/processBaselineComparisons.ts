import {
    FilterState,
    DeploymentEntity,
    ExternalSourceEntity,
    InternetEntity,
    BaselineComparisonsResponse,
    SimulatedBaseline,
    SimulatedBaselineStatus,
    Properties,
} from 'Containers/Network/networkTypes';
import { filterLabels } from 'constants/networkFilterModes';

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
        return 'External Entities';
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
    baselines: BaselineComparisonsResponse['added'] | BaselineComparisonsResponse['removed'],
    filterState: FilterState,
    simulatedStatus: SimulatedBaselineStatus
): SimulatedBaseline[] {
    return baselines.reduce((acc, baseline) => {
        const { entity, properties } = baseline;
        return [...acc, ...processBaseline(entity, properties, filterState, simulatedStatus)];
    }, [] as SimulatedBaseline[]);
}

function processReconciledBaselines(
    baselines: BaselineComparisonsResponse['reconciled'],
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
    { added, removed, reconciled }: BaselineComparisonsResponse,
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
