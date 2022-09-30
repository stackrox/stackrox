import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'policies'; // omit to minimize changed lines

describe('Config Management Entities (Policies)', () => {
    withAuth();

    it('should render the policies list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('policies');
    });

    it('should take you to a policy single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('policies');
        navigateToSingleEntityPage('policies');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('policies');
        navigateToSingleEntityPage('policies');
        hasCountWidgetsFor(['Deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        renderListAndSidePanel('policies');
        navigateToSingleEntityPage('policies');
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('policies');
        navigateToSingleEntityPage('policies');
        hasTabsFor(['deployments']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        const entitiesKey2 = 'deployments';

        context('Page', () => {
            renderListAndSidePanel('policies');
            navigateToSingleEntityPage('policies');
            pageEntityCountMatchesTableRows('policies', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('policies');
            sidePanelEntityCountMatchesTableRows('policies', entitiesKey2);
        });
    });
});
