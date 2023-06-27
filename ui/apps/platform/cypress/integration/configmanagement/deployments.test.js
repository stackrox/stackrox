import withAuth from '../../helpers/basicAuth';

import {
    clickEntityTableRowThatHasLinkInColumn,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    hasCountWidgetsFor,
    hasRelatedEntityFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyTableLinkToSidePanelTable,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntityInSidePanel,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'deployments';

describe('Configuration Management Deployments', () => {
    withAuth();

    it('should render the deployments list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should go from table link to images table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'images');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'clusters');
    });

    it('should take you to a deployments single when the "navigate away" button is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should show the related cluster, namespace, and service account widgets', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasRelatedEntityFor('Cluster');
        hasRelatedEntityFor('Namespace');
        hasRelatedEntityFor('Service Account');
    });

    it('should have the correct count widgets for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Images', 'Secrets']);
    });

    it('should have the correct tabs for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['images', 'secrets']);
    });

    it('should click on the images count widget in the entity page and show the images tab', () => {
        const columnIndexForImages = 6;
        clickEntityTableRowThatHasLinkInColumn(entitiesKey, columnIndexForImages);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('images', 'entityList');
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

    describe('should go to secrets table from widget link', () => {
        const entitiesKey2 = 'secrets';

        it('in single page', () => {
            verifyWidgetLinkToTableFromSinglePage(entitiesKey, entitiesKey2);
        });

        it('in side panel', () => {
            verifyWidgetLinkToTableFromSidePanel(entitiesKey, entitiesKey2);
        });
    });
});
