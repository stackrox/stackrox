import withAuth from '../../helpers/basicAuth';

import {
    clickEntityTableRowThatHasLinkInColumn,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    clickOnSingleEntityInTable,
    hasCountWidgetsFor,
    hasRelatedEntityFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyTableLinkToSidePanelTable,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntityInSidePanel,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'namespaces';

describe('Configuration Management Namespaces', () => {
    withAuth();

    it('should render the namespaces list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should render the namespaces list and open the side panel with the clicked cluster value', () => {
        clickOnSingleEntityInTable(entitiesKey, 'clusters');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'clusters');
    });

    it('should take you to a namespace single when the "navigate away" button is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should show the related cluster widget', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasRelatedEntityFor('Cluster');
    });

    it('should have the correct count widgets for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Deployments', 'Secrets', 'Images']);
    });

    it('should click on the secrets count widget in the entity page and show the secrets tab', () => {
        const columnIndexForSecrets = 5;
        clickEntityTableRowThatHasLinkInColumn(entitiesKey, columnIndexForSecrets);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('secrets', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments', 'secrets', 'images']);
    });

    describe('should go to deployments table from widget link', () => {
        const entitiesKey2 = 'deployments';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to secrets table from widget link', () => {
        const entitiesKey2 = 'secrets';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    describe('should go to images table from widget link', () => {
        const entitiesKey2 = 'images';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });

    it('should go from table link to serviceaccounts table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'serviceaccounts');
    });

    it('should go from table link to roles table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'roles');
    });
});
