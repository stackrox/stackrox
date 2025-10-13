import entityTypes from 'constants/entityTypes';

import { getEntityState } from 'test-utils/workflowUtils';

// system under test (SUT)
import { createOptions, getOption } from './workflowUtils';

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
});
