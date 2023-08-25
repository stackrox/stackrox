import withAuth from '../../helpers/basicAuth';

import {
    clickEntityTableRowThatHasLinkInColumn,
    clickOnCountWidget,
    hasCountWidgetsFor,
    hasTabsFor,
    navigateToSingleEntityPage,
    verifyTableLinkToSidePanelTable,
    verifyWidgetLinkToTableFromSidePanel,
    verifyWidgetLinkToTableFromSinglePage,
    visitConfigurationManagementEntityInSidePanel,
} from './ConfigurationManagement.helpers';

const entitiesKey = 'images';

describe('Configuration Management Images', () => {
    withAuth();

    it('should render the images list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
    });

    it('should go from table link to deployments table in side panel', () => {
        verifyTableLinkToSidePanelTable(entitiesKey, 'deployments');
    });

    it('should take you to a images single when the "navigate away" button is clicked', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should have the correct count widgets for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Deployments']);
    });

    it('should click on the deployments count widget in the entity page and show the deployments tab', () => {
        const columnIndexForDeployments = 4;
        clickEntityTableRowThatHasLinkInColumn(entitiesKey, columnIndexForDeployments);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Deployments']);
        clickOnCountWidget('deployments', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        visitConfigurationManagementEntityInSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['deployments']);
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

    // regression test for ROX-4543-crash-when-drilling-down-to-image-deployments
    it('should allow user to drill down from cluster to image to image-deployments', () => {
        visitConfigurationManagementEntityInSidePanel('clusters');
        clickOnCountWidget(entitiesKey, 'side-panel');
        cy.get(`[data-testid="side-panel"] .rt-tbody .rt-tr:last`).click({
            force: true,
        });
        clickOnCountWidget('deployments', 'entityList');

        // GraphQL error takes a while to show up, and just extending the cy.get timeout does not work with should-not
        cy.wait(1000);
        cy.get('[data-testid="graphql-error"]').should('not.exist');
    });
});
