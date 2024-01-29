import withAuth from '../../helpers/basicAuth';

import {
    clickEntityTableRowThatHasLinkInColumn,
    clickOnCountWidget,
    clickOnSingularEntityWidgetInSidePanel,
    hasCountWidgetsFor,
    hasRelatedEntityFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntities,
    visitConfigurationManagementEntityInSidePanel,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'secrets';

describe('Configuration Management Secrets', () => {
    withAuth();

    it('should render the secrets list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should render the deployments link and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntities(entitiesKey);

        cy.get('.rt-tbody .rt-tr').find(`.rt-td a[data-testid='deployment']`).eq(0).click();
        cy.get('[data-testid="side-panel"] [data-testid="panel-header"]:contains("deployment")');
    });

    it('should click on the cluster entity widget in the side panel and match the header ', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        clickOnSingularEntityWidgetInSidePanel(entitiesKey, 'clusters');
    });

    it('should take you to a secrets single when the "navigate away" button is clicked', () => {
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
        hasCountWidgetsFor(['Deployments']);
    });

    it('should have the correct tabs for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        const columnIndexForDeployments = 6;
        clickEntityTableRowThatHasLinkInColumn(entitiesKey, columnIndexForDeployments);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('deployments', 'entityList');
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
});
