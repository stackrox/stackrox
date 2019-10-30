import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';

describe('Vuln Management Dashboard Page', () => {
    withAuth();
    // TODO re-enable the following test after bug ROX-3571 is fixed
    it.skip('should show same number of policies between the tile and the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(0)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then(value => {
                const numPolicies = value;
                cy.get(selectors.tileLinks)
                    .eq(0)
                    .click();
                cy.get(`[data-test-id="panel"] [data-test-id="panel-header"]`)
                    .invoke('text')
                    .then(panelHeaderText => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numPolicies, 10));
                    });
            });
    });

    it.skip('should show same number of cves between the tile and the cves list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(1)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then(value => {
                const numCves = value;
                cy.get(selectors.tileLinks)
                    .eq(1)
                    .click();
                cy.get(`[data-test-id="panel"] [data-test-id="panel-header"]`)
                    .invoke('text')
                    .then(panelHeaderText => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numCves, 10));
                    });
            });
    });

    it('should properly navigate to the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.tileLinks)
            .eq(0)
            .click();
        cy.url().should('contain', url.list.policies);
    });

    it('should properly navigate to the clusters list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('clusters')).click();
        cy.url().should('contain', url.list.clusters);
    });

    it('should properly navigate to the namespaces list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('namespaces')).click();
        cy.url().should('contain', url.list.namespaces);
    });

    it('should properly navigate to the deployments list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('deployments')).click();
        cy.url().should('contain', url.list.deployments);
    });

    it('should properly navigate to the images list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('images')).click();
        cy.url().should('contain', url.list.images);
    });
});
