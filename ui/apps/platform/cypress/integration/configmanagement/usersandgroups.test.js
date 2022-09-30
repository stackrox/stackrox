import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    entityListCountMatchesTableLinkCount,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'subjects'; // omit to minimize changed lines

describe('Config Management Entities (Subjects - Users & Groups', () => {
    withAuth();

    it('should render the users & groups list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('subjects');
    });

    it('should open the side panel to show the same number of roles when the roles link is clicked', () => {
        entityListCountMatchesTableLinkCount('subjects', 'roles', /^\d+ Roles?$/);
    });

    it('should take you to a subject single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subjects');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subjects');
        hasCountWidgetsFor(['Roles']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subjects');
        hasTabsFor(['roles']);
    });

    it('should click on the roles count widget in the entity page and show the roles tab', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subjects');
        clickOnCountWidget('roles', 'entityList');
    });

    it('should have the same number of Roles in the count widget as in the Roles table', () => {
        const entitiesKey2 = 'roles';

        context('Page', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subjects');
            pageEntityCountMatchesTableRows('subjects', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('subjects');
            sidePanelEntityCountMatchesTableRows('subjects', entitiesKey2);
        });
    });
});
