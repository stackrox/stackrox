import entityTypes from 'constants/entityTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import useCases from 'constants/useCaseTypes';
import WorkflowEntity from 'utils/WorkflowEntity';
import { WorkflowState } from 'utils/WorkflowState';

// system under test (SUT)
import { createOptions, getOption, shouldUseOriginalCase } from './workflowUtils';

describe('workflowUtils', () => {
    describe('createOptions', () => {
        it('should return a list of workflow state menu option objects', () => {
            const isSidePanelOpen = true;
            const entityWorkflowState = getEntityState(isSidePanelOpen);

            const availableEntityTypes = [entityTypes.IMAGE, entityTypes.COMPONENT];

            const menuOptions = createOptions(availableEntityTypes, entityWorkflowState);

            expect(menuOptions).toEqual([
                {
                    label: 'images',
                    link: '/main/configmanagement/images',
                },
                {
                    label: 'components',
                    link: '/main/configmanagement/components',
                },
            ]);
        });
    });

    describe('getOption', () => {
        it('should return a single workflow state menu option objects', () => {
            const isSidePanelOpen = true;
            const entityWorkflowState = getEntityState(isSidePanelOpen);

            const entityType = entityTypes.COMPONENT;

            const menuOption = getOption(entityType, entityWorkflowState);

            expect(menuOption).toEqual({
                label: 'components',
                link: '/main/configmanagement/components',
            });
        });
    });

    describe('shouldUseOriginalCase', () => {
        it('should be true for a image with a name', () => {
            const entityName = 'wordpress';
            const entityType = entityTypes.IMAGE;

            const showInOriginalCase = shouldUseOriginalCase(entityName, entityType);

            expect(showInOriginalCase).toBe(true);
        });

        it('should be true for a component with a name', () => {
            const entityName = 'ncurses';
            const entityType = entityTypes.COMPONENT;

            const showInOriginalCase = shouldUseOriginalCase(entityName, entityType);

            expect(showInOriginalCase).toBe(true);
        });

        it('should be false for an entity with a name, which is not an image nor compoennt', () => {
            const entityName = 'ncurses';
            const entityType = entityTypes.CLUSTER;

            const showInOriginalCase = shouldUseOriginalCase(entityName, entityType);

            expect(showInOriginalCase).toBe(false);
        });
    });
});

const entityId1 = '1234';
const entityId2 = '5678';

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
        useCases.CONFIG_MANAGEMENT,
        stateStack,
        searchParamValues,
        sortParamValues,
        pagingParamValues
    );
}
