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

const entitiesKey = 'controls';

describe('Configuration Management Controls', () => {
    withAuth();

    it('should render the controls list and open the side panel when a row is clicked', () => {
        visitConfigurationManagementDashboard();

        // This and the following tests assumes that scan results are available
        interactAndWaitForConfigurationManagementScan(() => {
            cy.get('[data-testid="scan-button"]').click();
        });

        renderListAndSidePanel(entitiesKey);
    });

    it('should take you to a control single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasCountWidgetsFor(['Nodes']);
    });

    // ROX-13028: skip pending investigation why sometimes 0 nodes for control, therefore widget is disabled.
    it.skip('should click on the nodes count widget in the entity page and show the nodes tab', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        clickOnCountWidget('nodes', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel(entitiesKey);
        navigateToSingleEntityPage(entitiesKey);
        hasTabsFor(['nodes']);
    });

    describe('should have same number in nodes table as in count widget', () => {
        const entitiesKey2 = 'nodes';

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

    it('should show no failing nodes in the control findings section of a passing control', () => {
        visitConfigurationManagementEntities(entitiesKey);

        cy.get(configManagementSelectors.tableNextPage).click();
        cy.get(configManagementSelectors.tableCells)
            .contains(controlStatus.pass)
            .eq(0)
            .click({ force: true });
        cy.get(configManagementSelectors.failingNodes).should('have.length', 0);
    });

    // ROX-13028: skip pending investigation why sometimes table has 0 failing nodes.
    it.skip('should show failing nodes in the control findings section of a failing control', () => {
        visitConfigurationManagementEntities(entitiesKey);

        cy.get(configManagementSelectors.tableCells)
            .contains(controlStatus.fail)
            .eq(0)
            .click({ force: true });
        cy.get(configManagementSelectors.failingNodes).should('not.have.length', 0);
    });
});
