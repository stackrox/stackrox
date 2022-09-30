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

// const entitiesKey = 'roles'; // omit to minimize changed lines

describe('Config Management Entities (Roles)', () => {
    withAuth();

    it('should render the roles list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel('roles');
    });

    it('should take you to a roles single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('roles');
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('roles');
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('roles');
        hasCountWidgetsFor(['Users & Groups', 'Service Accounts']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('roles');
        navigateToSingleEntityPage('roles');
        hasTabsFor(['users and groups', 'service accounts']);
    });

    it('should have the same number of Users & Groups in the count widget as in the Users & Groups table', () => {
        const entitiesKey2 = 'subjects';

        context('Page', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('roles');
            pageEntityCountMatchesTableRows('roles', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('roles');
            sidePanelEntityCountMatchesTableRows('roles', entitiesKey2);
        });
    });

    it('should have the same number of Service Accounts in the count widget as in the Service Accounts table', () => {
        const entitiesKey2 = 'serviceaccounts';

        context('Page', () => {
            renderListAndSidePanel('roles');
            navigateToSingleEntityPage('roles');
            pageEntityCountMatchesTableRows('roles', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('roles');
            sidePanelEntityCountMatchesTableRows('roles', entitiesKey2);
        });
    });
});
