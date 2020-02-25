import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnRowEntity,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Subjects - Users & Groups', () => {
    withAuth();

    it('should render the users & groups list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('subjects');
    });

    it('should click on the roles link in the users & groups list and open the side panel with the roles list', () => {
        clickOnRowEntity('subjects', 'roles');
    });

    it('should take you to a subject single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subject');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subject');
        hasCountWidgetsFor(['Roles']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subject');
        hasTabsFor(['roles']);
    });

    it('should click on the roles count widget in the entity page and show the roles tab', () => {
        renderListAndSidePanel('subjects');
        navigateToSingleEntityPage('subject');
        clickOnCountWidget('roles', 'entityList');
    });

    it('should have the same number of Roles in the count widget as in the Roles table', () => {
        context('Page', () => {
            renderListAndSidePanel('subjects');
            navigateToSingleEntityPage('subject');
            pageEntityCountMatchesTableRows('Roles');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('subjects');
            sidePanelEntityCountMatchesTableRows('Roles');
        });
    });
});
