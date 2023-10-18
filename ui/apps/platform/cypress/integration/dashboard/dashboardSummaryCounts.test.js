import { resourceToAccess as resourceToAccessForAnalyst } from '../../fixtures/auth/mypermissionsForAnalyst.json';
import { resourceToAccess as resourceToAccessForNoAccess } from '../../fixtures/auth/mypermissionsNoAccess.json';

import withAuth from '../../helpers/basicAuth';
import {
    routeMatcherMapForSummaryCounts,
    visitMainDashboardWithStaticResponseForPermissions,
} from '../../helpers/main';

function getStaticResponseForAnalystWithoutResources(resources) {
    const resourceToAccess = { ...resourceToAccessForAnalyst };

    resources.forEach((resource) => {
        resourceToAccess[resource] = 'NO_ACCESS';
    });

    return { body: { resourceToAccess } };
}

function getStaticResponseForNoAccessExceptResources(resources) {
    const resourceToAccess = { ...resourceToAccessForNoAccess };

    resources.forEach((resource) => {
        resourceToAccess[resource] = 'READ_ACCESS';
    });

    return { body: { resourceToAccess } };
}

function getSummaryCountSelector(noun) {
    return `main section:first-child a.pf-c-button.pf-m-link:contains("${noun}")`;
}

const lastUpdatedSelector = 'main section:first-child div:contains("Last updated")';

describe('Dashboard SummaryCounts', () => {
    withAuth();

    it('should display 6 counts for Analyst role', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources([]),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 0 counts with no resources', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources([])
            // no request
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector).should('not.exist');
    });

    it('should display 5 counts without Cluster resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources(['Cluster']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Cluster resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources(['Cluster']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Node resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources(['Node']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Node resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources(['Node']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Alert resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources(['Alert']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Alert resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources(['Alert']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Deployment resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources(['Deployment']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Deployment resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources(['Deployment']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Image resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources(['Image']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Image resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources(['Image']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 5 counts without Secret resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForAnalystWithoutResources(['Secret']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster'));
        cy.get(getSummaryCountSelector('Node'));
        cy.get(getSummaryCountSelector('Violation'));
        cy.get(getSummaryCountSelector('Deployment'));
        cy.get(getSummaryCountSelector('Image'));
        cy.get(getSummaryCountSelector('Secret')).should('not.exist');
        cy.get(lastUpdatedSelector);
    });

    it('should display 1 count with only Secret resource', () => {
        visitMainDashboardWithStaticResponseForPermissions(
            getStaticResponseForNoAccessExceptResources(['Secret']),
            routeMatcherMapForSummaryCounts
        );

        cy.get(getSummaryCountSelector('Cluster')).should('not.exist');
        cy.get(getSummaryCountSelector('Node')).should('not.exist');
        cy.get(getSummaryCountSelector('Violation')).should('not.exist');
        cy.get(getSummaryCountSelector('Deployment')).should('not.exist');
        cy.get(getSummaryCountSelector('Image')).should('not.exist');
        cy.get(getSummaryCountSelector('Secret'));
        cy.get(lastUpdatedSelector);
    });
});
