import {
    renderListAndSidePanel,
    navigateToSingleEntityPage,
    hasCountWidgetsFor,
    clickOnCountWidget,
    hasTabsFor,
    pageEntityCountMatchesTableRows,
    sidePanelEntityCountMatchesTableRows,
} from '../../helpers/configWorkflowUtils';
import {
    url,
    selectors as configManagementSelectors,
    controlStatus,
} from '../../constants/ConfigManagementPage';
import withAuth from '../../helpers/basicAuth';
import { triggerScan } from '../../helpers/compliance';

describe('Config Management Entities (CIS controls)', () => {
    withAuth();

    it('should render the controls list and open the side panel when a row is clicked', () => {
        triggerScan(); // because tests assume that scan results are available

        renderListAndSidePanel('controls');
    });

    it('should take you to a control single when the "navigate away" button is clicked', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('control');
    });

    it('should have the correct count widgets for a single entity view', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('control');
        hasCountWidgetsFor(['Nodes']);
    });

    it('should click on the nodes count widget in the entity page and show the nodes tab', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('control');
        clickOnCountWidget('nodes', 'entityList');
    });

    it('should have the correct tabs for a single entity view', () => {
        renderListAndSidePanel('controls');
        navigateToSingleEntityPage('control');
        hasTabsFor(['nodes']);
    });

    it('should have the same number of Nodes in the count widget as in the Nodes table', () => {
        context('Page', () => {
            renderListAndSidePanel('controls');
            navigateToSingleEntityPage('control');
            pageEntityCountMatchesTableRows('Nodes');
        });

        context('Side Panel', () => {
            renderListAndSidePanel('controls');
            sidePanelEntityCountMatchesTableRows('Nodes');
        });
    });

    it('should show no failing nodes in the control findings section of a passing control', () => {
        cy.visit(url.list.controls);
        cy.wait(5000);
        cy.get(configManagementSelectors.tableNextPage).click();
        cy.get(configManagementSelectors.tableCells)
            .contains(controlStatus.pass)
            .eq(0)
            .click({ force: true });
        cy.get(configManagementSelectors.failingNodes).should('have.length', 0);
    });

    it('should show failing nodes in the control findings section of a failing control', () => {
        cy.visit(url.list.controls);
        cy.get(configManagementSelectors.tableCells)
            .contains(controlStatus.fail)
            .eq(0)
            .click({ force: true });
        cy.get(configManagementSelectors.failingNodes).should('not.have.length', 0);
    });
});
