import withAuth from '../../helpers/basicAuth';

import {
    visitConfigurationManagementEntityInSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    entityListCountMatchesTableLinkCount,
    hasTabsFor,
    hasRelatedEntityFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'deployments';

describe('Configuration Management Deployments', () => {
    withAuth();

    it('should render the deployments list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should open the side panel to show the same number of secrets when the secrets link is clicked', () => {
        entityListCountMatchesTableLinkCount(entitiesKey, 'secrets', /\d+ secrets?$/);
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
        visitConfigurationManagementEntityInSidePanel(entitiesKey, 'collector');
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('images', 'entityList');
    });

    describe('should have same number in images table as in count widget', () => {
        const entitiesKey2 = 'images';

        it('of page', () => {
            visitConfigurationManagementEntityInSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            visitConfigurationManagementEntityInSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });

    describe('should have same number in secrets table as in count widget', () => {
        const entitiesKey2 = 'secrets';

        it('of page', () => {
            visitConfigurationManagementEntityInSidePanel(entitiesKey);
            navigateToSingleEntityPage(entitiesKey);
            pageEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });

        it('of side panel', () => {
            visitConfigurationManagementEntityInSidePanel(entitiesKey);
            sidePanelEntityCountMatchesTableRows(entitiesKey, entitiesKey2);
        });
    });
});
