import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import WorkflowStateMgr, { WorkflowEntity, WorkflowState } from './WorkflowStateManager';

const entityId1 = '1234';
const entityId2 = '5678';
const entityId3 = '1111';
const useCase = useCases.CONFIG_MANAGEMENT;

const searchParamValues = {
    [searchParams.page]: {
        sk1: 'v1',
        sk2: 'v2'
    },
    [searchParams.sidePanel]: {
        sk3: 'v3',
        sk4: 'v4'
    }
};

const sortParamValues = {
    [sortParams.page]: entityTypes.CLUSTER,
    [sortParams.sidePanel]: entityTypes.DEPLOYMENT
};

const pagingParamValues = {
    [pagingParams.page]: 1,
    [pagingParams.sidePanel]: 2
};

function getEntityState(isSidePanelOpen) {
    const stateStack = [new WorkflowEntity(entityTypes.CLUSTER, entityId1)];
    if (isSidePanelOpen) {
        stateStack.push(new WorkflowEntity(entityTypes.DEPLOYMENT));
        stateStack.push(new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2));
    }

    return new WorkflowState(
        useCase,
        stateStack,
        searchParamValues,
        sortParamValues,
        pagingParamValues
    );
}

function getListState(isSidePanelOpen) {
    const stateStack = [new WorkflowEntity(entityTypes.CLUSTER)];
    if (isSidePanelOpen) stateStack.push(new WorkflowEntity(entityTypes.CLUSTER, entityId1));

    return new WorkflowState(
        useCase,
        stateStack,
        searchParamValues,
        sortParamValues,
        pagingParamValues
    );
}

describe('WorkflowStateManager', () => {
    it('resets current state based on given parameters', () => {
        const workflowStateMgr = new WorkflowStateMgr(getEntityState());
        workflowStateMgr.reset(useCase, entityTypes.DEPLOYMENT, entityId2);

        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId2 }
        ]);
    });

    it('Removes sidepanel params state', () => {
        // in list
        let workflowStateMgr = new WorkflowStateMgr(getListState());
        workflowStateMgr.removeSidePanelParams();

        expect(workflowStateMgr.workflowState.stateStack).toEqual([{ t: entityTypes.CLUSTER }]);

        // in entity
        workflowStateMgr = new WorkflowStateMgr(getEntityState(true));
        workflowStateMgr.removeSidePanelParams();

        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.DEPLOYMENT }
        ]);
    });
    it('pushes a list onto the stack related to current workflow state', () => {
        // entity page
        let workflowStateMgr = new WorkflowStateMgr(getEntityState());
        workflowStateMgr.pushList(entityTypes.NAMESPACE);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE }
        ]);

        // list page
        workflowStateMgr = new WorkflowStateMgr(getListState(true));
        workflowStateMgr.pushList(entityTypes.NAMESPACE);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE }
        ]);
    });
    it('pushes a list onto the stack and overflows stack properly', () => {
        // parent relationship + 1 (entity page)
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
            new WorkflowEntity(entityTypes.SECRET, entityId3)
        ]);
        let workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.DEPLOYMENT);

        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.DEPLOYMENT }
        ]);

        // parent relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.NAMESPACE),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
            new WorkflowEntity(entityTypes.SECRET),
            new WorkflowEntity(entityTypes.SECRET, entityId3)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.DEPLOYMENT);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.DEPLOYMENT }
        ]);

        // matches relationship + 1 (entity page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.SECRET, entityId3),
            new WorkflowEntity(entityTypes.CLUSTER, entityId2)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.NODE);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 },
            { t: entityTypes.NODE }
        ]);

        // matches relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.SECRET),
            new WorkflowEntity(entityTypes.SECRET, entityId3),
            new WorkflowEntity(entityTypes.CLUSTER, entityId2)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.NODE);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 },
            { t: entityTypes.NODE }
        ]);

        // contained inferred relationship + 1 (entity page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.IMAGE, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.SERVICE_ACCOUNT);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.SERVICE_ACCOUNT }
        ]);

        // contained inferred relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.CLUSTER),
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.SERVICE_ACCOUNT);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.SERVICE_ACCOUNT }
        ]);
    });

    it('pushes an entity of a list by id onto the stack', () => {
        const workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT)
        ]);
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushListItem(entityId1);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId1 }
        ]);
    });
    it('replaces an entity of a list by pushing id onto the stack', () => {
        const workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1)
        ]);
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushListItem(entityId2);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId2 }
        ]);
    });
    it('pushes a related entity to the stack', () => {
        const workflowStateMgr = new WorkflowStateMgr(getEntityState());
        workflowStateMgr.pushRelatedEntity(entityTypes.POLICY, entityId2);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.POLICY, i: entityId2 }
        ]);
    });
    it('pushes a related entity onto the stack and overflows stack properly', () => {
        // parents relationship + 1 (entity page)
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId3)
        ]);
        let workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId2);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 }
        ]);

        // parents relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId3)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId2);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 }
        ]);

        // matches relationship + 1 (entity page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.POLICY, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId1);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 }
        ]);

        // matches relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.POLICY, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId1);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 }
        ]);

        // contained inferred relationship + 1
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.SUBJECT, entityId1),
            new WorkflowEntity(entityTypes.ROLE, entityId1)
        ]);
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.SUBJECT, entityId2);
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.SUBJECT, i: entityId2 }
        ]);
    });

    it('pops the last entity off of the stack', () => {
        const workflowStateMgr = new WorkflowStateMgr(getEntityState());
        workflowStateMgr.pop();
        expect(workflowStateMgr.workflowState.stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 }
        ]);
    });

    it('sets search state for page', () => {
        const workflowStateMgr = new WorkflowStateMgr(getEntityState());
        workflowStateMgr.setSearch({ testKey: 'testVal' });
        expect(workflowStateMgr.workflowState.search[searchParams.page]).toEqual({
            testKey: 'testVal'
        });
        expect(workflowStateMgr.workflowState.search[searchParams.sidePanel]).toEqual(
            searchParamValues[searchParams.sidePanel]
        );
    });

    it('sets search state for sidePanel', () => {
        const workflowStateMgr = new WorkflowStateMgr(getListState(true));
        workflowStateMgr.setSearch({ testKey: 'testVal' });
        expect(workflowStateMgr.workflowState.search[searchParams.sidePanel]).toEqual({
            testKey: 'testVal'
        });
        expect(workflowStateMgr.workflowState.search[searchParams.page]).toEqual(
            searchParamValues[searchParams.page]
        );
    });
});
