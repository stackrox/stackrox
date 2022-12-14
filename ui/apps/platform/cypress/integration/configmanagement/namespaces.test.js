import withAuth from '../../helpers/basicAuth';

import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    clickOnSingleEntityInTable,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
    entityListCountMatchesTableLinkCount,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'namespaces';

describe('Configuration Management Namespaces', () => {
    withAuth();

    it('should render the namespaces list and open the side panel when a row is clicked', () => {
        renderListAndSidePanel(entitiesKey);
    });

    it('should render the namespaces list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntityInTable(entitiesKey, 'clusters');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        renderListAndSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'clusters');
    });

    it('should take you to a namespace single when the "navigate away" button is clicked', () => {
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
        hasCountWidgetsFor(['Deployments', 'Secrets', 'Images']);
    });

    it('should click on the secrets count widget in the entity page and show the secrets tab', () => {
        renderListAndSidePanel(entitiesKey, 'stackrox');
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('secrets', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments', 'secrets', 'images']);
    });

    describe('should have same number in deployments table as in count widget', () => {
        const entitiesKey2 = 'deployments';

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

    describe('should have same number in secrets table as in count widget', () => {
        const entitiesKey2 = 'secrets';

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

    describe('should have same number in images table as in count widget', () => {
        const entitiesKey2 = 'images';

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

    it('should open the side panel to show the same number of Service Accounts when the Service Accounts link is clicked', () => {
        entityListCountMatchesTableLinkCount(
            entitiesKey,
            'serviceaccounts',
            /^\d+ Service Accounts?$/
        );
    });

    it('should open the side panel to show the same number of Roles when the Roles link is clicked', () => {
        entityListCountMatchesTableLinkCount(entitiesKey, 'roles', /^\d+ Roles?$/);
    });
});
