import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';

function validateTopRiskyEntities(entityName) {
    cy.visit(url.dashboard);
    cy.get(selectors.topRiskyItems.select.value).should(
        'contain',
        'Top risky deployments by CVE count & CVSS score'
    );
    cy.get(selectors.topRiskyItems.select.input)
        .first()
        .click();
    cy.get(selectors.topRiskyItems.select.options)
        .contains(`Top risky ${entityName} by CVE count & CVSS score`)
        .click();
    cy.get(selectors.topRiskyItems.select.value).should(
        'contain',
        `Top risky ${entityName} by CVE count & CVSS score`
    );
    cy.get(selectors.getWidget(`Top risky ${entityName} by CVE count & CVSS score`))
        .find(selectors.viewAllButton)
        .click();
    cy.wait(500);
    if (entityName === 'clusters') cy.url().should('contain', url.list.clusters);
    else if (entityName === 'images') cy.url().should('contain', url.list.images);
    else if (entityName === 'namespaces') cy.url().should('contain', url.list.namespaces);
    else if (entityName === 'deployments') cy.url().should('contain', url.list.deployments);
}

describe('Vuln Management Dashboard Page', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

    withAuth();

    it('should show same number of policies between the tile and the policies list', () => {
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

    it('should show same number of cves between the tile and the cves list', () => {
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

    // @TODO: add check that changing entity type re-displays the loader
    //   not reliable to test without a good way to mock GraphQL responses
    it('"Top Riskiest <entities>" widget should start with a loading indicator', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Top risky deployments by CVE count & CVSS score'))
            .find(selectors.widgetBody)
            .invoke('text')
            .then(bodyText => {
                expect(bodyText).to.contain('Loading');
            });
    });

    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Top Riskiest Images" widget\'s "View All" button should take you to the images list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Top Riskiest Images'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.images);
    });

    // TODO  change the sort param checked for, if a more desirable sort becomes available from the API
    //   see https://stack-rox.atlassian.net/browse/ROX-4295 for details
    it('clicking the "Frequently Violated Policies" widget\'s "View All" button should take you to the policies list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Frequently Violated Policies'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.policies);

        // check sort requested
        cy.url().should('contain', 'sort[0][id]=Severity');
        cy.url().should('contain', 'sort[0][desc]=true');
    });

    // TODO  add a check that there is a sort param on the link URL for sorting by the widget's appropriate sort
    it('clicking the "Recently Detected Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Recently Detected Vulnerabilities'))
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
    it('clicking the "Clusters With Most K8s & Istio Vulnerabilities" widget\'s "View All" button should take you to the clusters list', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.getWidget('Clusters With Most K8s & Istio Vulnerabilities'))
            .find(selectors.viewAllButton)
            .click();
        cy.url().should('contain', url.list.clusters);
    });

    it('clicking the "Top risky deployments by CVE count & CVSS score" widget\'s "View All" button should take you to the deployments list', () => {
        validateTopRiskyEntities('deployments');
    });

    it('clicking the "Top risky namespaces by CVE count & CVSS score" widget\'s "View All" button should take you to the namespaces list', () => {
        validateTopRiskyEntities('namespaces');
    });

    it('clicking the "Top risky images by CVE count & CVSS score" widget\'s "View All" button should take you to the images list', () => {
        validateTopRiskyEntities('images');
    });

    it('clicking the "Top risky clusters by CVE count & CVSS score" widget\'s "View All" button should take you to the clusters list', () => {
        validateTopRiskyEntities('clusters');
    });
});
