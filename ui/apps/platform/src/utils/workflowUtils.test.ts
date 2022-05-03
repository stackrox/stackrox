import entityTypes from 'constants/entityTypes';

import { getEntityState } from 'test-utils/workflowUtils';

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
