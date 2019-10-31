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
    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Top Riskiest Images" widget\'s "View All" button should take you to the images list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Top Riskiest Images'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.images);
    });
    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Frequently Violated Policies" widget\'s "View All" button should take you to the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Frequently Violated Policies'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.policies);
    });
    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Most Recent Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Most Recent Vulnerabilities'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.cves);
    });
    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Most Common Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Most Common Vulnerabilities'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.cves);
    });
    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Deployments With Most Severe Policy Violations" widget\'s "View All" button should take you to the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Deployments With Most Severe Policy Violations'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.deployments);
    });
    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Clusters With Most K8s Vulnerabilities" widget\'s "View All" button should take you to the clusters list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Clusters With Most K8s Vulnerabilities'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.clusters);
    });
});
