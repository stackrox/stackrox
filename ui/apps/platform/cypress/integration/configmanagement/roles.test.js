import withAuth from '../../helpers/basicAuth';

import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'roles';

describe('Configuration Management Roles', () => {
    withAuth();

    it('should render the roles list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel(entitiesKey);
    });

    it('should take you to a roles single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should show the related cluster widget', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Users & Groups', 'Service Accounts']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['users and groups', 'service accounts']);
    });

    describe('should have same number in users and groups table as in count widget', () => {
        const entitiesKey2 = 'subjects';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in service accounts table as in count widget', () => {
        const entitiesKey2 = 'serviceaccounts';

        it('of page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });
});
