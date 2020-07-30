import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

describe('Config Management Entities (Roles)', () => {
    withAuth();

    it('should render the roles list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('roles');
    });

    it('should take you to a roles single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('role');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('role');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('role');
        hasCountWidgetsFor(['Users & Groups', 'Service Accounts']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('role');
        hasTabsFor(['users and groups', 'service accounts']);
    });

    it('should have the same number of Users & Groups in the count widget as in the Users & Groups table', () => {
        context('Page', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            pageEntityCountMatchesTableRows('Users & Groups');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('roles');
            sidePanelEntityCountMatchesTableRows('Users & Groups');
        });
    });

    it('should have the same number of Service Accounts in the count widget as in the Service Accounts table', () => {
        context('Page', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('role');
            pageEntityCountMatchesTableRows('Service Accounts');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('roles');
            sidePanelEntityCountMatchesTableRows('Service Accounts');
        });
    });
});
