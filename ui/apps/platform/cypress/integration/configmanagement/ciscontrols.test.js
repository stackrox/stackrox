import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
    interactAndWaitForConfigurationManagementScan,
    visitConfigurationManagementDashboard,
    visitConfigurationManagementEntities,
} from '../../helpers/configWorkflowUtils';
import {
    selectors as configManagementSelectors,
    controlStatus,
} from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';

// const entitiesKey = 'controls'; // omit to minimize changed lines

describe('Config Management Entities (CIS controls)', () => {
    withAuth();

    it('should render the controls list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementDashboard();

        // This and the following tests assumes that scan results are available
        interactAndWaitForConfigurationManagementScan(() => {
            cy.get('[data-testid="scan-button"]').click();
        });

        renderListAndSidePanel('controls');
    });

    it('should take you to a control single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('controls');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('controls');
        hasCountWidgetsFor(['Nodes']);
    });

    it('should click on the nodes count widget in the entity page and show the nodes tab', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('controls');
        clickOnCountWidget('nodes', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('controls');
        hasTabsFor(['nodes']);
    });

    it('should have the same number of Nodes in the count widget as in the Nodes table', () => {
        const entitiesKey2 = 'nodes';

        context('Page', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('controls');
            pageEntityCountMatchesTableRows('controls', entitiesKey2);
        });

        context('Side Panel', () => {
            renderListAndSidePanel('controls');
            sidePanelEntityCountMatchesTableRows('controls', entitiesKey2);
        });
    });

    it('should show no failing nodes in the control findings section of a passing control', () => {
        visitConfigurationManagementEntities('controls');

        cy.get(configManagementSelectors.tableNextPage).click();
        cy.get(configManagementSelectors.tableCells)
            .contains(controlStatus.pass)
            .eq(0)
            .click({ force: true });
        cy.get(configManagementSelectors.failingNodes).should('have.length', 0);
    });

    it('should show failing nodes in the control findings section of a failing control', () => {
        visitConfigurationManagementEntities('controls');

        cy.get(configManagementSelectors.tableCells)
            .contains(controlStatus.fail)
            .eq(0)
            .click({ force: true });
        cy.get(configManagementSelectors.failingNodes).should('not.have.length', 0);
    });
});
