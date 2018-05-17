import { url as violationsUrl, selectors as ViolationsPageSelectors } from './pages/ViolationsPage';
import * as api from './apiEndpoints';
import selectors from './pages/SearchPage';

describe('Violations page', () => {
    const setUpAlertsByPolicies = () => {
        cy.server();
        cy.fixture('alerts/alertsByPolicies.json').as('alertsByPolicies');
        cy.route('GET', api.alerts.alertsByPolicies, '@alertsByPolicies').as('alertsByPolicies');
        cy.wait('@alertsByPolicies');
    };

    const setUpAlertsByPolicyId = () => {
        cy.fixture('alerts/alertsByPolicyId.json').as('alertsByPolicyId');
        cy.route('GET', api.alerts.alertsByPolicyId, '@alertsByPolicyId').as('alertsByPolicyId');
        cy.wait('@alertsByPolicyId');
    };

    it('should select item in nav bar', () => {
        cy.visit(violationsUrl);
        cy.get(ViolationsPageSelectors.navLink).should('have.class', 'bg-primary-600');
    });

    it('should have violations in table', () => {
        setUpAlertsByPolicies();
        cy.get(ViolationsPageSelectors.rows).should('have.length', 2);
    });

    it('should show side panel with panel header', () => {
        setUpAlertsByPolicies();
        cy.get(ViolationsPageSelectors.firstTableRow).click();
        setUpAlertsByPolicyId();
        cy.get(ViolationsPageSelectors.panelHeader).should('have.text', 'abcd');
    });

    it('should have cluster column in table', () => {
        cy.get(ViolationsPageSelectors.clusterTableHeader).should('be.visible');
    });

    it('should click on first row in side panel and launch modal', () => {
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.get(ViolationsPageSelectors.modal).should('be.visible');
    });

    it('should have cluster field in modal', () => {
        cy.get(ViolationsPageSelectors.clusterFieldInModal).should('be.visible');
    });

    it('should close the side panel on search filter', () => {
        cy.visit(violationsUrl);
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy.get('.side-panel').should('not.be.visible');
    });
});
