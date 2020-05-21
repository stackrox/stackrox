import { url, selectors } from '../constants/RiskPage';
import withAuth from '../helpers/basicAuth';

describe('Risk Flow', () => {
    withAuth();

    beforeEach(() => {
        cy.visit(url);
    });

    it('visa processor should be the top riskiest deployment', () => {
        cy.get(selectors.table.rows)
            .eq(0)
            .get(selectors.table.cells)
            .eq(0)
            .invoke('text')
            .then((value) => {
                expect(value).to.equal('visa-processor');
            });
    });

    it('visa-processor should have generated violations and other risk attributes', () => {
        cy.get(selectors.table.rows).eq(0).click();
        cy.get(selectors.collapsible.card)
            .find(selectors.collapsible.body)
            .children()
            .should('not.be.empty');
    });

    it('visa-processor should have static deployment details', () => {
        cy.get(selectors.table.rows).eq(0).click();
        cy.get(selectors.panelTabs.deploymentDetails).click();
        cy.get(selectors.collapsible.card)
            .eq(0)
            .find(selectors.collapsible.body)
            .should('contain', 'Deployment Type:')
            .should('contain', 'Namespace')
            .should('contain', 'Replicas:')
            .should('contain', 'Cluster:');
    });

    it('visa-processor should have flagged processes', () => {
        cy.get(selectors.table.rows).eq(0).click();
        cy.get(selectors.panelTabs.processDiscovery).click();
        cy.get(selectors.suspiciousProcesses);
    });
});
