import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import WorkflowStateMgr from './WorkflowStateManager';

const entityId1 = '1234';
const entityId2 = '5678';
const entityId3 = '1111';
const useCase = useCases.CONFIG_MANAGEMENT;

describe('WorkflowStateManager', () => {
    it('resets current state based on given parameters', () => {
        const workflowState = {
            useCase,
            stateStack: [{ t: entityTypes.NAMESPACE }, { t: entityTypes.NAMESPACE, i: entityId1 }]
        };
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.base(entityTypes.DEPLOYMENT, entityId2, useCase);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId2 }]
        });
    });
    it('pushes a list onto the stack related to current workflow state', () => {
        // entity page
        let workflowState = {
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId1 }]
        };
        let workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.NAMESPACE);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId1 }, { t: entityTypes.NAMESPACE }]
        });

        // list page
        workflowState = {
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT }, { t: entityTypes.DEPLOYMENT, i: entityId1 }]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.NAMESPACE);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.NAMESPACE }
            ]
        });
    });
    it('pushes a list onto the stack and overflows stack properly', () => {
        // parent relationship + 1 (entity page)
        let workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.NAMESPACE, i: entityId2 },
                { t: entityTypes.SECRET, i: entityId3 }
            ]
        };
        let workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.DEPLOYMENT);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.SECRET, i: entityId3 }, { t: entityTypes.DEPLOYMENT }]
        });

        // parent relationship + 1 (list page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.NAMESPACE },
                { t: entityTypes.NAMESPACE, i: entityId2 },
                { t: entityTypes.SECRET },
                { t: entityTypes.SECRET, i: entityId3 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.DEPLOYMENT);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.SECRET, i: entityId3 }, { t: entityTypes.DEPLOYMENT }]
        });

        // matches relationship + 1 (entity page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.SECRET, i: entityId3 },
                { t: entityTypes.CLUSTER, i: entityId2 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.NODE);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.CLUSTER, i: entityId2 }, { t: entityTypes.NODE }]
        });

        // matches relationship + 1 (list page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.SECRET },
                { t: entityTypes.SECRET, i: entityId3 },
                { t: entityTypes.CLUSTER, i: entityId2 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.NODE);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.CLUSTER, i: entityId2 }, { t: entityTypes.NODE }]
        });

        // contained inferred relationship + 1 (entity page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.CLUSTER, i: entityId1 },
                { t: entityTypes.IMAGE, i: entityId2 },
                { t: entityTypes.DEPLOYMENT, i: entityId3 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.SERVICE_ACCOUNT);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId3 },
                { t: entityTypes.SERVICE_ACCOUNT }
            ]
        });

        // contained inferred relationship + 1 (list page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.CLUSTER },
                { t: entityTypes.CLUSTER, i: entityId1 },
                { t: entityTypes.IMAGE },
                { t: entityTypes.IMAGE, i: entityId2 },
                { t: entityTypes.DEPLOYMENT, i: entityId3 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushList(entityTypes.SERVICE_ACCOUNT);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId3 },
                { t: entityTypes.SERVICE_ACCOUNT }
            ]
        });
    });
    it('pushes an entity of a list by id onto the stack', () => {
        const workflowState = {
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT }]
        };
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushListItem(entityId1);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId1 }]
        });
    });
    it('replaces an entity of a list by pushing id onto the stack', () => {
        const workflowState = {
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId1 }]
        };
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushListItem(entityId2);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId2 }]
        });
    });
    it('pushes a related entity to the stack', () => {
        const workflowState = {
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId1 }]
        };
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.POLICY, entityId2);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.POLICY, i: entityId2 }
            ]
        });
    });
    it('pushes a related entity onto the stack and overflows stack properly', () => {
        // parents relationship + 1 (entity page)
        let workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.IMAGE, i: entityId1 },
                { t: entityTypes.DEPLOYMENT, i: entityId2 },
                { t: entityTypes.NAMESPACE, i: entityId3 }
            ]
        };
        let workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId2);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.CLUSTER, i: entityId2 }]
        });

        // parents relationship + 1 (list page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.IMAGE },
                { t: entityTypes.IMAGE, i: entityId1 },
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId2 },
                { t: entityTypes.NAMESPACE, i: entityId3 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId2);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.CLUSTER, i: entityId2 }]
        });

        // matches relationship + 1 (entity page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.POLICY, i: entityId2 },
                { t: entityTypes.DEPLOYMENT, i: entityId3 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId1);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.CLUSTER, i: entityId1 }]
        });

        // matches relationship + 1 (list page)
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.POLICY, i: entityId2 },
                { t: entityTypes.DEPLOYMENT },
                { t: entityTypes.DEPLOYMENT, i: entityId3 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.CLUSTER, entityId1);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.CLUSTER, i: entityId1 }]
        });

        // contained inferred relationship + 1
        workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.CLUSTER, i: entityId1 },
                { t: entityTypes.SUBJECT, i: entityId1 },
                { t: entityTypes.ROLE, i: entityId1 }
            ]
        };
        workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushRelatedEntity(entityTypes.SUBJECT, entityId2);
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.SUBJECT, i: entityId2 }]
        });
    });
    it('pops the last entity off of the stack', () => {
        const workflowState = {
            useCase,
            stateStack: [
                { t: entityTypes.DEPLOYMENT, i: entityId1 },
                { t: entityTypes.NAMESPACE, i: entityId2 }
            ]
        };
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pop();
        expect(workflowStateMgr.workflowState).toEqual({
            useCase,
            stateStack: [{ t: entityTypes.DEPLOYMENT, i: entityId1 }]
        });
    });
});
