import { resourceToAccess as resourceToAccessForAnalyst } from '../../fixtures/auth/mypermissionsForAnalyst.json';
import { resourceToAccess as resourceToAccessForNoAccess } from '../../fixtures/auth/mypermissionsNoAccess.json';

import withAuth from '../../helpers/basicAuth';
import { visitMainDashboardWithStaticResponseForPermissions } from '../../helpers/main';

function getDataForAnalystWithoutResources(resources) {
    const resourceToAccess = { ...resourceToAccessForAnalyst };

    resources.forEach((resource) => {
        resourceToAccess[resource] = 'NO_ACCESS';
    });

    return { resourceToAccess };
}

function getDataForNoAccessExceptResources(resources) {
    const resourceToAccess = { ...resourceToAccessForNoAccess };

    resources.forEach((resource) => {
        resourceToAccess[resource] = 'READ_ACCESS';
    });

    return { resourceToAccess };
}

function getSummaryCountSelector(noun) {
    return `main section:first-child a.pf-c-button.pf-m-link:contains("${noun}")`;
}

const lastUpdatedSelector = 'main section:first-child div:contains("Last updated")';

describe('Dashboard SummaryCounts', () => {
    withAuth();

    it('should display 6 counts for Analyst role', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources([]),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 0 counts with no resources', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources([]),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector).should('not.exist');
    });

    it('should display 5 counts without Cluster resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources(['Cluster']),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Cluster resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources(['Cluster']),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Node resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources(['Node']),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Node resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources(['Node']),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Alert resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources(['Alert']),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Alert resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources(['Alert']),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Deployment resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources(['Deployment']),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it.only('should display 1 count with only Deployment resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources(['Deployment']),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Image resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources(['Image']),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Image resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources(['Image']),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Secret resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForAnalystWithoutResources(['Secret']),
        });

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Secret resource', () => {
        visitMainDashboardWithStaticResponseForPermissions({
            body: getDataForNoAccessExceptResources(['Secret']),
        });

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });
});
