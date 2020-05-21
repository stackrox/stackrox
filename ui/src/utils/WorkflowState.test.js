import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import WorkflowEntity from './WorkflowEntity';
import { WorkflowState } from './WorkflowState';

const entityId1 = '1234';
const entityId2 = '5678';
const entityId3 = '1111';
const useCase = useCases.CONFIG_MANAGEMENT;

const searchParamValues = {
    [searchParams.page]: {
        sk1: 'v1',
        sk2: 'v2',
    },
    [searchParams.sidePanel]: {
        sk3: 'v3',
        sk4: 'v4',
    },
};

const sortParamValues = {
    [sortParams.page]: entityTypes.CLUSTER,
    [sortParams.sidePanel]: entityTypes.DEPLOYMENT,
};

const pagingParamValues = {
    [pagingParams.page]: 1,
    [pagingParams.sidePanel]: 2,
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
    it('clears current state on current use case', () => {
        expect(getEntityState().clear().stateStack).toEqual([]);
    });

    it('resets current state based on given parameters', () => {
        expect(
            getEntityState().reset(useCase, entityTypes.DEPLOYMENT, entityId2).stateStack
        ).toEqual([{ t: entityTypes.DEPLOYMENT, i: entityId2 }]);
    });

    describe('removeSidePanelParams', () => {
        it('Removes sidepanel params state for a list', () => {
            const isSidePanelOpen = true;
            expect(getListState(isSidePanelOpen).removeSidePanelParams().stateStack).toEqual([
                { t: entityTypes.CLUSTER },
            ]);
        });

        it('Removes sidepanel params state, and preserves the list search', () => {
            const isSidePanelOpen = true;
            const sidepanelStateWithSearch = getListState(isSidePanelOpen);

            const newState = sidepanelStateWithSearch.removeSidePanelParams();

            expect(newState.search[searchParams.page]).toEqual(
                searchParamValues[searchParams.page]
            );
            expect(newState.search[searchParams.sidePanel]).toBeFalsy();
        });

        it('Removes sidepanel params state, and preserves the list sort', () => {
            const isSidePanelOpen = true;
            const sidepanelStateWithSearch = getListState(isSidePanelOpen);

            const newState = sidepanelStateWithSearch.removeSidePanelParams();

            expect(newState.sort[sortParams.page]).toEqual(sortParamValues[sortParams.page]);
            expect(newState.sort[sortParams.sidePanel]).toBeFalsy();
        });

        it('Removes sidepanel params state, and preserves the list pagination', () => {
            const isSidePanelOpen = true;
            const sidepanelStateWithSearch = getListState(isSidePanelOpen);

            const newState = sidepanelStateWithSearch.removeSidePanelParams();

            expect(newState.paging[pagingParams.page]).toEqual(
                pagingParamValues[pagingParams.page]
            );
            expect(newState.paging[pagingParams.sidePanel]).toBeFalsy();
        });

        it('Removes sidepanel params state for an entity', () => {
            const isSidePanelOpen = true;
            expect(getEntityState(isSidePanelOpen).removeSidePanelParams().stateStack).toEqual([
                { t: entityTypes.CLUSTER, i: entityId1 },
                { t: entityTypes.DEPLOYMENT },
            ]);
        });
    });

    describe('getSingleAncestorOfType', () => {
        it('finds an ancestor entity type in the state stack when present', () => {
            // arrange
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.CLUSTER),
                new WorkflowEntity(entityTypes.CLUSTER, entityId1),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
                new WorkflowEntity(entityTypes.POLICY),
                new WorkflowEntity(entityTypes.POLICY, entityId3),
            ]);

            // act
            const hasDeploymentAncestor = workflowState.getSingleAncestorOfType(
                entityTypes.DEPLOYMENT
            );

            // assert
            expect(hasDeploymentAncestor).toEqual(
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2)
            );
        });

        it('does not find an ancestor entity type in the state stack when not present', () => {
            // arrange
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.CLUSTER),
                new WorkflowEntity(entityTypes.CLUSTER, entityId1),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
                new WorkflowEntity(entityTypes.POLICY),
                new WorkflowEntity(entityTypes.POLICY, entityId3),
            ]);

            // act
            const hasDeploymentAncestor = workflowState.getSingleAncestorOfType(
                entityTypes.NAMESPACE
            );

            // assert
            expect(hasDeploymentAncestor).toBe(null);
        });
    });

    it('pushes a list onto the stack related to current workflow state', () => {
        // dashboard
        const workflowState = new WorkflowState(useCase, []);
        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.NAMESPACE },
        ]);

        // entity page
        expect(getEntityState().pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE },
        ]);

        // list page
        expect(getListState(true).pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE },
        ]);
    });

    describe('WorkflowState parent relationship logic', () => {
        it('pushes a parent relationship onto the stack', () => {
            // parent relationship (from entity page)
            let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            ]);

            expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.NAMESPACE },
            ]);

            // parent relationship (from list page)
            workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            ]);
            expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.NAMESPACE },
            ]);
        });

        it('pushes a parent relationship onto the stack and overflows from list page', () => {
            // parent relationship + 1 (list page)
            let workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
                new WorkflowEntity(entityTypes.SECRET),
                new WorkflowEntity(entityTypes.SECRET, entityId3),
            ]);
            expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
                { t: entityTypes.SECRET, i: entityId3 },
                { t: entityTypes.DEPLOYMENT },
            ]);

            // parent relationship + 1 (list page)
            workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.CLUSTER),
                new WorkflowEntity(entityTypes.CLUSTER, entityId1),
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId2),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3),
            ]);
            expect(workflowState.pushList(entityTypes.SERVICE_ACCOUNT).stateStack).toEqual([
                { t: entityTypes.DEPLOYMENT, i: entityId3 },
                { t: entityTypes.SERVICE_ACCOUNT },
            ]);

            // deployments -> dep -> cluster -> namespaces (should nav away)
            workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
                new WorkflowEntity(entityTypes.CLUSTER, entityId2),
            ]);
            expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
                { t: entityTypes.CLUSTER, i: entityId2 },
                { t: entityTypes.NAMESPACE },
            ]);
        });

        it('pushes a parent relationship onto the stack and overflows from entity page', () => {
            // parent relationship + 1 (entity page)
            let workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
                new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
                new WorkflowEntity(entityTypes.SECRET, entityId3),
            ]);

            expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
                { t: entityTypes.SECRET, i: entityId3 },
                { t: entityTypes.DEPLOYMENT },
            ]);

            // parent relationship + 1 (entity page)
            workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.CLUSTER, entityId1),
                new WorkflowEntity(entityTypes.IMAGE, entityId2),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3),
            ]);
            expect(workflowState.pushList(entityTypes.SERVICE_ACCOUNT).stateStack).toEqual([
                { t: entityTypes.DEPLOYMENT, i: entityId3 },
                { t: entityTypes.SERVICE_ACCOUNT },
            ]);
        });
    });

    describe('WorkflowState matches relationship logic', () => {
        it('pushes a matches relationship onto the stack', () => {
            // matches relationship (entity page)
            let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            ]);
            expect(
                workflowState.pushRelatedEntity(entityTypes.POLICY, entityId2).stateStack
            ).toEqual([
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.POLICY, i: entityId2 },
            ]);

            // matches relationship (list page)
            workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            ]);
            expect(
                workflowState.pushRelatedEntity(entityTypes.POLICY, entityId2).stateStack
            ).toEqual([
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.POLICY, i: entityId2 },
            ]);
        });

        it('images -> image -> deployment should not nav away', () => {
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId1),
            ]);
            expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
                { t: entityTypes.IMAGE },
                { t: entityTypes.IMAGE, i: entityId1 },
                { t: entityTypes.DEPLOYMENT },
            ]);
        });

        it('cves -> cve -> deployment should not nav away (from table count link as well)', () => {
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.CVE),
                new WorkflowEntity(entityTypes.CVE, entityId1),
            ]);
            expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
                { t: entityTypes.CVE },
                { t: entityTypes.CVE, i: entityId1 },
                { t: entityTypes.DEPLOYMENT },
            ]);
        });

        it('components -> images link in table count should not nav away', () => {
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.COMPONENT),
                new WorkflowEntity(entityTypes.COMPONENT, entityId1),
            ]);
            expect(workflowState.pushList(entityTypes.IMAGE).stateStack).toEqual([
                { t: entityTypes.COMPONENT },
                { t: entityTypes.COMPONENT, i: entityId1 },
                { t: entityTypes.IMAGE },
            ]);
        });
    });

    it('pushes a matches relationship onto the stack and overflows stack properly', () => {
        // matches relationship + 1 (entity page)
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.SECRET, entityId3),
        ]);
        const newWorkflowState = workflowState.pushList(entityTypes.NAMESPACE);
        expect(newWorkflowState.stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.NAMESPACE },
        ]);

        // matches relationship + 1 (list page)
        workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
            new WorkflowEntity(entityTypes.SECRET),
            new WorkflowEntity(entityTypes.SECRET, entityId3),
        ]);
        expect(workflowState.pushList(entityTypes.NAMESPACE).stateStack).toEqual([
            { t: entityTypes.SECRET, i: entityId3 },
            { t: entityTypes.NAMESPACE },
        ]);
    });

    it('pushes a contains relationship onto the stack', () => {
        // contained relationship (entity page)
        let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.IMAGE, entityId2),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId3),
        ]);
        const newWorkflowState = workflowState.pushList(entityTypes.CVE);
        expect(newWorkflowState.stateStack).toEqual([
            { t: entityTypes.IMAGE, i: entityId2 },
            { t: entityTypes.COMPONENT },
            { t: entityTypes.COMPONENT, i: entityId3 },
            { t: entityTypes.CVE },
        ]);

        // contained relationship (list page)
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId2),
        ]);
        expect(workflowState.pushList(entityTypes.CVE).stateStack).toEqual([
            { t: entityTypes.IMAGE },
            { t: entityTypes.IMAGE, i: entityId1 },
            { t: entityTypes.COMPONENT },
            { t: entityTypes.COMPONENT, i: entityId2 },
            { t: entityTypes.CVE },
        ]);

        // drilling down from cluster to leaf
        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CLUSTER),
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.NAMESPACE),
            new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3),
        ]);
        expect(workflowState.pushList(entityTypes.COMPONENT).stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE },
            { t: entityTypes.NAMESPACE, i: entityId2 },
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.COMPONENT },
        ]);
        expect(workflowState.pushList(entityTypes.CVE).stateStack).toEqual([
            { t: entityTypes.CLUSTER },
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.NAMESPACE },
            { t: entityTypes.NAMESPACE, i: entityId2 },
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId3 },
            { t: entityTypes.CVE },
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
            new WorkflowEntity(entityTypes.CVE, entityId3),
        ]);
        const newWorkflowState = workflowState.pushList(entityTypes.DEPLOYMENT);
        expect(newWorkflowState.stateStack).toEqual([
            { t: entityTypes.CVE, i: entityId3 },
            { t: entityTypes.DEPLOYMENT },
        ]);
    });

    it('pushes an entity of a list by id onto the stack', () => {
        const workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT),
        ]);
        expect(workflowState.pushListItem(entityId1).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT },
            { t: entityTypes.DEPLOYMENT, i: entityId1 },
        ]);
    });

    it('replaces an entity of a list by pushing id onto the stack', () => {
        const workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId1),
        ]);
        expect(workflowState.pushListItem(entityId2).stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId2 },
        ]);
    });

    it('pushes a related entity to the stack', () => {
        // dashboard
        const workflowState = new WorkflowState(useCase, []);
        expect(workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId2).stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId2 },
        ]);

        // entity page
        expect(
            getEntityState().pushRelatedEntity(entityTypes.POLICY, entityId2).stateStack
        ).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
            { t: entityTypes.POLICY, i: entityId2 },
        ]);
    });

    describe('WorkflowState pushRelatedEntity overflow logic', () => {
        it('overflows stack properly when pushing a related entity onto parent in stack', () => {
            // parents relationship + 1 (entity page)
            let workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.IMAGE, entityId1),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
                new WorkflowEntity(entityTypes.NAMESPACE, entityId3),
            ]);
            expect(
                workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId2).stateStack
            ).toEqual([{ t: entityTypes.CLUSTER, i: entityId2 }]);

            // parents relationship + 1 (list page)
            workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId1),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
                new WorkflowEntity(entityTypes.NAMESPACE, entityId3),
            ]);
            expect(
                workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId2).stateStack
            ).toEqual([{ t: entityTypes.CLUSTER, i: entityId2 }]);
        });

        it('overflows stack properly when pushing a related entity onto matches in stack', () => {
            // matches relationship + 1 (entity page)
            let workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.NAMESPACE, entityId1),
                new WorkflowEntity(entityTypes.POLICY, entityId2),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3),
            ]);
            expect(
                workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId1).stateStack
            ).toEqual([{ t: entityTypes.CLUSTER, i: entityId1 }]);

            // matches relationship + 1 (list page)
            workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, entityId1),
                new WorkflowEntity(entityTypes.POLICY, entityId2),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId3),
            ]);
            expect(
                workflowState.pushRelatedEntity(entityTypes.CLUSTER, entityId1).stateStack
            ).toEqual([{ t: entityTypes.CLUSTER, i: entityId1 }]);
        });

        it('overflows stack properly when pushing a duplicate entity onto stack', () => {
            // duplicate entity type on stack (list page)
            let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.CVE),
                new WorkflowEntity(entityTypes.CVE, entityId1),
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId2),
            ]);
            expect(workflowState.pushRelatedEntity(entityTypes.CVE, entityId3).stateStack).toEqual([
                { t: entityTypes.CVE, i: entityId3 },
            ]);

            // duplicate entity type on stack (entity page)
            workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
                new WorkflowEntity(entityTypes.CVE, entityId1),
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId2),
            ]);
            expect(workflowState.pushRelatedEntity(entityTypes.CVE, entityId3).stateStack).toEqual([
                { t: entityTypes.CVE, i: entityId3 },
            ]);
        });
    });

    it('pushes a duplicate entity onto the stack and overflows stack properly', () => {
        let workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId3),
        ]);
        expect(workflowState.pushList(entityTypes.DEPLOYMENT).stateStack).toEqual([
            { t: entityTypes.IMAGE, i: entityId3 },
            { t: entityTypes.DEPLOYMENT },
        ]);

        workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, [
            new WorkflowEntity(entityTypes.CLUSTER, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId3),
            new WorkflowEntity(entityTypes.COMPONENT),
            new WorkflowEntity(entityTypes.COMPONENT, entityId1),
        ]);
        expect(workflowState.pushList(entityTypes.IMAGE).stateStack).toEqual([
            { t: entityTypes.COMPONENT, i: entityId1 },
            { t: entityTypes.IMAGE },
        ]);
    });

    it('clears pagination when overflowing stack into a list', () => {
        const workflowState = new WorkflowState(
            useCases.VULN_MANAGEMENT,
            [
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId1),
                new WorkflowEntity(entityTypes.COMPONENT),
                new WorkflowEntity(entityTypes.COMPONENT, entityId2),
                new WorkflowEntity(entityTypes.CVE),
                new WorkflowEntity(entityTypes.CVE, entityId3),
            ],
            {},
            {},
            2
        );
        expect(workflowState.pushList(entityTypes.DEPLOYMENT).paging).not.toEqual(2);
    });

    it('does not clear pagination when pushing a list to the stack that does not overflow', () => {
        const workflowState = new WorkflowState(
            useCases.VULN_MANAGEMENT,
            [
                new WorkflowEntity(entityTypes.CLUSTER),
                new WorkflowEntity(entityTypes.NAMESPACE, entityId2),
            ],
            {},
            {},
            2
        );
        expect(workflowState.pushList(entityTypes.IMAGE).paging).toEqual(2);
    });

    it('pops the last entity off of the stack', () => {
        expect(getEntityState().pop().stateStack).toEqual([
            { t: entityTypes.CLUSTER, i: entityId1 },
        ]);
    });

    it('sets search state for page', () => {
        const newState = getEntityState().setSearch({ testKey: 'testVal' });
        expect(newState.search[searchParams.page]).toEqual({
            testKey: 'testVal',
        });
        expect(newState.search[searchParams.sidePanel]).toEqual(
            searchParamValues[searchParams.sidePanel]
        );
    });

    it('sets search state for sidePanel', () => {
        const newState = getListState(true).setSearch({ testKey: 'testVal' });

        expect(newState.search[searchParams.sidePanel]).toEqual({
            testKey: 'testVal',
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
            new WorkflowEntity(entityTypes.POLICY),
        ]);
        expect(workflowState.getEntityContext()).toEqual({
            [entityTypes.CLUSTER]: entityId1,
            [entityTypes.DEPLOYMENT]: entityId2,
        });
    });

    it('skims stack properly when slimming side panel for external link generation', () => {
        // skims to latest entity page
        let workflowState = new WorkflowState(useCase, [
            new WorkflowEntity(entityTypes.IMAGE),
            new WorkflowEntity(entityTypes.IMAGE, entityId1),
            new WorkflowEntity(entityTypes.DEPLOYMENT),
            new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
        ]);
        expect(workflowState.getSkimmedStack().stateStack).toEqual([
            { t: entityTypes.DEPLOYMENT, i: entityId2 },
        ]);

        // skims to latest entity page + related entity list
        workflowState = new WorkflowState(
            useCase,
            [
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId1),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
            ],
            searchParamValues,
            sortParamValues,
            pagingParamValues
        );

        const skimmedWorkflowState = workflowState.getSkimmedStack();
        expect(skimmedWorkflowState.stateStack).toEqual([
            { t: entityTypes.IMAGE, i: entityId1 },
            { t: entityTypes.DEPLOYMENT },
        ]);
        expect(skimmedWorkflowState.search[searchParams.page]).toEqual(
            searchParamValues[searchParams.sidePanel]
        );
        expect(skimmedWorkflowState.sort[sortParams.page]).toEqual(
            sortParamValues[sortParams.sidePanel]
        );
        expect(skimmedWorkflowState.paging[pagingParams.page]).toEqual(
            pagingParamValues[pagingParams.sidePanel]
        );
    });

    describe('getCurrentEntityType', () => {
        it('returns the type of the last (current) entity on the state stack', () => {
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.IMAGE),
                new WorkflowEntity(entityTypes.IMAGE, entityId1),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, entityId2),
            ]);

            expect(workflowState.getCurrentEntityType()).toEqual(entityTypes.DEPLOYMENT);
        });

        it('returns the type of the only entity on the state stack', () => {
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.IMAGE),
            ]);

            expect(workflowState.getCurrentEntityType()).toEqual(entityTypes.IMAGE);
        });

        it('returns null when there is nothing on the state stack', () => {
            const workflowState = new WorkflowState(useCase, []);

            expect(workflowState.getCurrentEntityType()).toEqual(null);
        });
    });

    describe('isBaseList', () => {
        it('should return true when the top-level in the state stack is the entity list specified, with no child selected', () => {
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.COMPONENT),
            ]);

            const actual = workflowState.isBaseList(entityTypes.COMPONENT);

            expect(actual).toEqual(true);
        });

        it('should return false when the top-level in the state stack is not the entity list specified', () => {
            const workflowState = new WorkflowState(useCase, [new WorkflowEntity(entityTypes.CVE)]);

            const actual = workflowState.isBaseList(entityTypes.COMPONENT);

            expect(actual).toEqual(false);
        });
    });

    describe('isPreceding', () => {
        it('should return true when the preceding entity type of leaf state is the given entity', () => {
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.CVE, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.COMPONENT),
            ]);

            const actual = workflowState.isPreceding(entityTypes.CVE);

            expect(actual).toEqual(true);
        });

        it('should return true when the preceding entity type of leaf state is given entity, alternate test', () => {
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.COMPONENT),
            ]);

            const actual = workflowState.isPreceding(entityTypes.DEPLOYMENT);

            expect(actual).toEqual(true);
        });

        it('should return false when the preceding entity type of leaf state is NOT the given entity', () => {
            const workflowState = new WorkflowState(useCase, [
                new WorkflowEntity(entityTypes.CLUSTER, '4321-dcba'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
            ]);

            const actual = workflowState.isPreceding(entityTypes.CVE);

            expect(actual).toEqual(false);
        });
    });
});
