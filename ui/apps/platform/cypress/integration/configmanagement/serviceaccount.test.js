import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnSingularEntityWidgetInSidePanel,
    clickOnSingleEntityInTable,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import withAuth from '../../helpers/basicAuth';

const entitiesKey = 'serviceaccounts';

describe('Config Management Entities (Service Accounts)', () => {
    withAuth();

    it('should render the service accounts list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel(entitiesKey);
    });

    it('should click on the namespace entity widget in the side panel and match the header', () => {
        renderListAndSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'namespaces');
    });

    it('should render the service list and open the side panel with the clicked namespace value', () => {
        clickOnSingleEntityInTable(entitiesKey, 'namespaces');
    });

    it('should take you to a service account single when the "navigate away" button is clicked', () => {
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
        hasCountWidgetsFor(['Deployments', 'Roles']);
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments', 'roles']);
    });

    it('should have the same number of Deployments in the count widget as in the Deployments table', () => {
        const entitiesKey2 = 'deployments';

        context('Page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    it('should have the same number of Roles in the count widget as in the Roles table', () => {
        const entitiesKey2 = 'roles';

        context('Page', () => {
            renderListAndSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });
});
