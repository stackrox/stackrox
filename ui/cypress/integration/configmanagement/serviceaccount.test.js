import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnEntityWidget,
    clickOnSingleEntity,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Service Accounts)', () => {
    withAuth();

    it('should render the service accounts list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('serviceAccounts');
    });

    it('should click on the namespace entity widget in the side panel and match the header', () => {
        renderListAndSidePanel('serviceAccounts');
        clickOnEntityWidget('namespace', 'side-panel');
    });

    it('should render the service list and open the side panel with the clicked namespace value', () => {
        clickOnSingleEntity('serviceAccounts', 'namespace');
    });

    it('should take you to a service account single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('serviceAccounts');
        navigateToSingleEntityPage('serviceAccount');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('serviceAccounts');
        navigateToSingleEntityPage('serviceAccount');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('serviceAccounts');
        navigateToSingleEntityPage('serviceAccount');
        hasCountWidgetsFor(['Deployments', 'Roles']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('serviceAccounts');
        navigateToSingleEntityPage('serviceAccount');
        hasTabsFor(['deployments', 'roles']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        context('Page', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            pageEntityCountMatchesTableRows('Deployments');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('serviceAccounts');
            sidePanelEntityCountMatchesTableRows('Deployments');
        });
    });

    it('should have the same number of Roles in the count widget as in the Roles table', () => {
        context('Page', () => {
            renderListAndSidePanel('serviceAccounts');
            navigateToSingleEntityPage('serviceAccount');
            pageEntityCountMatchesTableRows('Roles');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('serviceAccounts');
            sidePanelEntityCountMatchesTableRows('Roles');
        });
    });
});
