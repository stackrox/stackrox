import * as api from '../../constants/apiEndpoints';
import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { visitVulnerabilityManagementDashboard } from '../../helpers/vulnmanagement/entities';
import { hasFeatureFlag } from '../../helpers/features';

function validateTopRiskyEntities(entityName) {
    visitVulnerabilityManagementDashboard();
    cy.get(selectors.topRiskyItems.select.value).should(
        'contain',
        'Top risky deployments by CVE count & CVSS score'
    );
    cy.get(selectors.topRiskyItems.select.input).first().click();
    cy.get(selectors.topRiskyItems.select.options)
        .contains(`Top risky ${entityName} by CVE count & CVSS score`)
        .click();
    cy.get(selectors.topRiskyItems.select.value).should(
        'contain',
        `Top risky ${entityName} by CVE count & CVSS score`
    );
    cy.intercept('POST', api.vulnMgmt.graphqlEntities(entityName)).as('entities');
    cy.get(selectors.getWidget(`Top risky ${entityName} by CVE count & CVSS score`))
        .find(selectors.viewAllButton)
        .click();
    cy.wait('@entities');
    cy.location('pathname').should('eq', url.list[entityName]);
}

describe('Vuln Management Dashboard Page', () => {
    withAuth();

    it('should show same number of policies between the tile and the policies list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.tileLinks)
            .eq(0)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then((value) => {
                const numPolicies = value;
                cy.get(selectors.tileLinks).eq(0).click();
                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numPolicies, 10));
                    });
            });
    });

    // TODO: update CVE links to CVE tables checks, for VM Updates
    it.skip('should show same number of cves between the tile and the cves list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.tileLinks)
            .eq(1)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then((value) => {
                const numCves = value;
                cy.get(selectors.tileLinks).eq(1).click();
                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numCves, 10));
                    });
            });
    });

    it('should show same number of images between the tile and the images list', () => {
        visitVulnerabilityManagementDashboard();

        const tileToCheck = hasFeatureFlag('ROX_FRONTEND_VM_UPDATES') ? 2 : 3;
        cy.log({ tileToCheck });
        cy.get(selectors.tileLinks, { timeout: 8000 })
            .eq(tileToCheck)
            .find(selectors.tileLinkValue)
            .invoke('text')
            .then((value) => {
                const numImages = value;
                cy.get(selectors.tileLinks).eq(tileToCheck).click();
                cy.get(`[data-testid="panel"] [data-testid="panel-header"]`)
                    .invoke('text')
                    .then((panelHeaderText) => {
                        expect(parseInt(panelHeaderText, 10)).to.equal(parseInt(numImages, 10));
                    });
            });
    });

    it('should properly navigate to the policies list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.tileLinks).eq(0).click();
        cy.location('pathname').should('eq', url.list.policies);
    });

    it('should properly navigate to the clusters list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('clusters')).click();
        cy.location('pathname').should('eq', url.list.clusters);
    });

    it('should properly navigate to the namespaces list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('namespaces')).click();
        cy.location('pathname').should('eq', url.list.namespaces);
    });

    it('should properly navigate to the deployments list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.applicationAndInfrastructureDropdown).click();
        cy.get(selectors.getMenuListItem('deployments')).click();
        cy.location('pathname').should('eq', url.list.deployments);
    });

    it('"Top Riskiest <entities>" widget should start with a loading indicator', () => {
        cy.visit(url.dashboard); // do not call visit helper because it waits on the requests
        cy.get(selectors.getWidget('Top risky deployments by CVE count & CVSS score'))
            .find(selectors.widgetBody)
            .invoke('text')
            .then((bodyText) => {
                expect(bodyText).to.contain('Loading');
            });
    });

    it('clicking the "Top Riskiest Images" widget\'s "View All" button should take you to the images list', () => {
        visitVulnerabilityManagementDashboard();
        cy.intercept('POST', api.vulnMgmt.graphqlEntities('images')).as('images');
        cy.get(selectors.getWidget('Top Riskiest Images')).find(selectors.viewAllButton).click();
        cy.wait('@images');
        cy.location('pathname').should('eq', url.list.images);
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Image%20Risk%20Priority&sort[0][desc]=false'
        );
    });

    // TODO  change the sort param checked for, if a more desirable sort becomes available from the API
    //   see https://stack-rox.atlassian.net/browse/ROX-4295 for details
    it('clicking the "Frequently Violated Policies" widget\'s "View All" button should take you to the policies list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.getWidget('Frequently Violated Policies'))
            .find(selectors.viewAllButton)
            .click();
        cy.location('pathname').should('eq', url.list.policies);
        cy.location('search').should('eq', '?sort[0][id]=Severity&sort[0][desc]=true');
    });

    it('clicking the "Recently Detected Image Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        visitVulnerabilityManagementDashboard();

        const titleToExpect = hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')
            ? 'Recently Detected Image Vulnerabilities'
            : 'Recently Detected Vulnerabilities';

        const urlToExpect = hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')
            ? url.list['image-cves']
            : url.list.cves;

        cy.get(selectors.getWidget(titleToExpect)).find(selectors.viewAllButton).click();
        cy.location('pathname').should('eq', urlToExpect);
        cy.location('search').should('eq', '?sort[0][id]=CVE%20Created%20Time&sort[0][desc]=true');
    });

    it('clicking the "Most Common Image Vulnerabilities" widget\'s "View All" button should take you to the CVEs list', () => {
        visitVulnerabilityManagementDashboard();

        const titleToExpect = hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')
            ? 'Most Common Image Vulnerabilities'
            : 'Most Common Vulnerabilities';

        const urlToExpect = hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')
            ? url.list['image-cves']
            : url.list.cves;

        cy.get(selectors.getWidget(titleToExpect)).find(selectors.viewAllButton).click();
        cy.location('pathname').should('eq', urlToExpect);
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Deployment%20Count&sort[0][desc]=true&sort[1][id]=CVSS&sort[1][desc]=true'
        );
    });

    it('clicking the "Deployments With Most Severe Policy Violations" widget\'s "View All" button should take you to the policies list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.getWidget('Deployments With Most Severe Policy Violations'))
            .find(selectors.viewAllButton)
            .click();
        cy.location('pathname').should('eq', url.list.deployments);
        cy.location('search').should('eq', '');
    });

    it('clicking the "Clusters With Most Orchestrator & Istio Vulnerabilities" widget\'s "View All" button should take you to the clusters list', () => {
        visitVulnerabilityManagementDashboard();
        cy.get(selectors.getWidget('Clusters With Most Orchestrator & Istio Vulnerabilities'))
            .find(selectors.viewAllButton)
            .click();
        cy.location('pathname').should('eq', url.list.clusters);
        cy.location('search').should('eq', '');
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
});
