import entityTypes from 'constants/entityTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import useCases from 'constants/useCaseTypes';
import WorkflowEntity from 'utils/WorkflowEntity';
import { WorkflowState } from 'utils/WorkflowState';

export const entityId1 = '1234';
export const entityId2 = '5678';
export const entityId3 = '1111';

export const searchParamValues = {
    [searchParams.page]: {
        sk1: 'v1',
        sk2: 'v2',
    },
    [searchParams.sidePanel]: {
        sk3: 'v3',
        sk4: 'v4',
    },
};

export const sortParamValues = {
    [sortParams.page]: entityTypes.CLUSTER,
    [sortParams.sidePanel]: entityTypes.DEPLOYMENT,
};

export const pagingParamValues = {
    [pagingParams.page]: 1,
    [pagingParams.sidePanel]: 2,
};

export function getEntityState(isSidePanelOpen) {
    const stateStack = [new WorkflowEntity(entityTypes.CLUSTER, entityId1)];
    if (isSidePanelOpen) {
        stateStack.push(new WorkflowEntity(entityTypes.DEPLOYMENT));
        stateStack.push(new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2));
    }

    return new WorkflowState(
        useCases.CONFIG_MANAGEMENT,
        stateStack,
        searchParamValues,
        sortParamValues,
        pagingParamValues
    );
}

export function getListState(isSidePanelOpen) {
    const stateStack = [new WorkflowEntity(entityTypes.CLUSTER)];
    if (isSidePanelOpen) {
        stateStack.push(new WorkflowEntity(entityTypes.CLUSTER, entityId1));
    }

    return new WorkflowState(
        useCases.CONFIG_MANAGEMENT,
        stateStack,
        searchParamValues,
        sortParamValues,
        pagingParamValues
    );
}
