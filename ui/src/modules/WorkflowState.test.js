import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import { WorkflowEntity, WorkflowState } from './WorkflowState';

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

describe('WorkflowState', () => {
    it('resets current state based on given parameters', () => {
        expect(
            getEntityState().reset(useCase, entityTypes.DEPLOYMENT, entityId2).stateStack
        ).toEqual([{ t: entityTypes.DEPLOYMENT, i: entityId2 }]);
    });

    it('Removes sidepanel params state', () => {
        // in list
        expect(getListState().removeSidePanelParams().stateStack).toEqual([
            { t: entityTypes.CLUSTER }
        ]);

        // in entity
        expect(getEntityState(true).removeSidePanelParams().stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.DEPLOYMENT }
        ]);
    });

    it('pushes a list onto the stack related to current workflow state', () => {
        // dashboard
        const workflowState = new WorkflowState(useCase, []);
        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.NAMESPACE }
        ]);

        // entity page
        expect(getEntityState().pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE }
        ]);

        // list page
        expect(getListState(true).pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE }
        ]);
    });

    it('pushes a parent relationship onto the stack', () => {
        // parent relationship (from entity page)
        let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1)
        ]);

        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId1 },
            { t: entityTypes.NAMESPACE }
        ]);

        // parent relationship (from list page)
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1)
        ]);
        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId1 },
            { t: entityTypes.NAMESPACE }
        ]);
    });

    it('pushes a parent relationship onto the stack and overflows stack properly', () => {
        // parent relationship + 1 (entity page)
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
            new WorkflowEntity(entityTypes.SECRET, entityId3)
        ]);

        expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
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
        expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.DEPLOYMENT }
        ]);

        // parent relationship + 1 (entity page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.IMAGE, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        expect(workflowState.pushList(entityTypes.SERVICE_ACCOUNT).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.SERVICE_ACCOUNT }
        ]);

        // parent relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.CLUSTER),
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        expect(workflowState.pushList(entityTypes.SERVICE_ACCOUNT).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.SERVICE_ACCOUNT }
        ]);

        // deployments -> dep -> cluster -> namespaces (should nav away)
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.CLUSTER, entityId2)
        ]);
        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 },
            { t: entityTypes.NAMESPACE }
        ]);
    });

    it('pushes a matches relationship onto the stack', () => {
        // matches relationship (entity page)
        let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1)
        ]);
        expect(workflowState.pushRelatedEntity(entityTypes.POLICY, entityId2).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId1 },
            { t: entityTypes.POLICY, i: entityId2 }
        ]);

        // matches relationship (list page)
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1)
        ]);
        expect(workflowState.pushRelatedEntity(entityTypes.POLICY, entityId2).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId1 },
            { t: entityTypes.POLICY, i: entityId2 }
        ]);

        // images -> image -> deployment should not nav away
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1)
        ]);
        expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
            { t: entityTypes.IMAGE },
            { t: entityTypes.IMAGE, i: entityId1 },
            { t: entityTypes.DEPLOYMENT }
        ]);

        // cves -> cve -> deployment should not nav away (from table count link as well)
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CVE),
            new WorkflowEntity(entityTypes.CVE, entityId1)
        ]);
        expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
            { t: entityTypes.CVE },
            { t: entityTypes.CVE, i: entityId1 },
            { t: entityTypes.DEPLOYMENT }
        ]);

        // components -> images link in table count shoudl not nav away
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId1)
        ]);
        expect(workflowState.pushList(entityTypes.IMAGE).stateStack).toEqual([
            { t: entityTypes.COMPONENT },
            { t: entityTypes.COMPONENT, i: entityId1 },
            { t: entityTypes.IMAGE }
        ]);
    });

    it('pushes a matches relationship onto the stack and overflows stack properly', () => {
        // matches relationship + 1 (entity page)
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.SECRET, entityId3)
        ]);
        const newWorkflowState = workflowState.pushList(entityTypes.NAMESPACE);
        expect(newWorkflowState.stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.NAMESPACE }
        ]);

        // matches relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.SECRET),
            new WorkflowEntity(entityTypes.SECRET, entityId3)
        ]);
        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.NAMESPACE }
        ]);
    });

    it('pushes a contains relationship onto the stack', () => {
        // contained relationship (entity page)
        let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.IMAGE, entityId2),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId3)
        ]);
        const newWorkflowState = workflowState.pushList(entityTypes.CVE);
        expect(newWorkflowState.stateStack).toEqual([
            { t: entityTypes.IMAGE, i: entityId2 },
            { t: entityTypes.COMPONENT },
            { t: entityTypes.COMPONENT, i: entityId3 },
            { t: entityTypes.CVE }
        ]);

        // contained relationship (list page)
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId2)
        ]);
        expect(workflowState.pushList(entityTypes.CVE).stateStack).toEqual([
            { t: entityTypes.IMAGE },
            { t: entityTypes.IMAGE, i: entityId1 },
            { t: entityTypes.COMPONENT },
            { t: entityTypes.COMPONENT, i: entityId2 },
            { t: entityTypes.CVE }
        ]);

        // drilling down from cluster to leaf
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CLUSTER),
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.NAMESPACE),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        expect(workflowState.pushList(entityTypes.COMPONENT).stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE },
            { t: entityTypes.NAMESPACE, i: entityId2 },
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.COMPONENT }
        ]);
        expect(workflowState.pushList(entityTypes.CVE).stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE },
            { t: entityTypes.NAMESPACE, i: entityId2 },
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.CVE }
        ]);
    });

    it('pushes a contains relationship onto the stack and overflows stack properly', () => {
        // drilling down from list to last leaf + 1
        const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId2),
            new WorkflowEntity(entityTypes.CVE),
            new WorkflowEntity(entityTypes.CVE, entityId3)
        ]);
        const newWorkflowState = workflowState.pushList(entityTypes.DEPLOYMENT);
        expect(newWorkflowState.stateStack).toEqual([
            { t: entityTypes.CVE, i: entityId3 },
            { t: entityTypes.DEPLOYMENT }
        ]);
    });

    it('pushes an entity of a list by id onto the stack', () => {
        const workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT)
        ]);
        expect(workflowState.pushListItem(entityId1).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId1 }
        ]);
    });

    it('replaces an entity of a list by pushing id onto the stack', () => {
        const workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1)
        ]);
        expect(workflowState.pushListItem(entityId2).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId2 }
        ]);
    });

    it('pushes a related entity to the stack', () => {
        // dashboard
        const workflowState = new WorkflowState(useCase, []);
        expect(workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId2).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 }
        ]);

        // entity page
        expect(
            getEntityState().pushRelatedEntity(entityTypes.POLICY, entityId2).stateStack
        ).toEqual([
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
        expect(workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId2).stateStack).toEqual([
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
        expect(workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId2).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 }
        ]);

        // matches relationship + 1 (entity page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.NAMESPACE, entityId1),
            new WorkflowEntity(entityTypes.POLICY, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        expect(workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId1).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 }
        ]);

        // matches relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.NAMESPACE),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId1),
            new WorkflowEntity(entityTypes.POLICY, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3)
        ]);
        expect(workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId1).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 }
        ]);
    });

    it('pushes a duplicate entity onto the stack and overflows stack properly', () => {
        let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId3)
        ]);
        expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
            { t: entityTypes.IMAGE, i: entityId3 },
            { t: entityTypes.DEPLOYMENT }
        ]);

        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId3),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId1)
        ]);
        expect(workflowState.pushList(entityTypes.IMAGE).stateStack).toEqual([
            { t: entityTypes.COMPONENT, i: entityId1 },
            { t: entityTypes.IMAGE }
        ]);
    });

    it('pops the last entity off of the stack', () => {
        expect(getEntityState().pop().stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 }
        ]);
    });

    it('sets search state for page', () => {
        const newState = getEntityState().setSearch({ testKey: 'testVal' });
        expect(newState.search[searchParams.page]).toEqual({
            testKey: 'testVal'
        });
        expect(newState.search[searchParams.sidePanel]).toEqual(
            searchParamValues[searchParams.sidePanel]
        );
    });

    it('sets search state for sidePanel', () => {
        const newState = getListState(true).setSearch({ testKey: 'testVal' });

        expect(newState.search[searchParams.sidePanel]).toEqual({
            testKey: 'testVal'
        });
        expect(newState.search[searchParams.page]).toEqual(searchParamValues[searchParams.page]);
    });

    it('generates correct entityContext', () => {
        let workflowState = new WorkflowState();
        expect(workflowState.getEntityContext()).toEqual({});

        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.CLUSTER),
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.POLICY)
        ]);
        expect(workflowState.getEntityContext()).toEqual({
            [entityTypes.CLUSTER]: entityId1,
            [entityTypes.DEPLOYMENT]: entityId2
        });
    });

    it('skims stack properly when slimming side panel for external link generation', () => {
        // skims to latest entity page
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2)
        ]);
        expect(workflowState.getSkimmedStack().stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId2 }
        ]);

        // skims to latest entity page + related entity list
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT)
        ]);
        expect(workflowState.getSkimmedStack().stateStack).toEqual([
            { t: entityTypes.IMAGE, i: entityId1 },
            { t: entityTypes.DEPLOYMENT }
        ]);
    });
});
