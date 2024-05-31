import withAuth from '../../helpers/basicAuth';
import { hasOrchestratorFlavor } from '../../helpers/features';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import {
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

const entitiesKey = 'cluster-cves';

describe('Vulnerability Management Cluster (Platform) CVEs', () => {
    withAuth();

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasTableColumnHeadings([
            '', // checkbox
            '', // hidden
            'CVE',
            'Type',
            'Fixable',
            'CVSS Score',
            'Env. Impact',
            'Impact Score',
            'Entities',
            'Published',
            '', // hidden
        ]);
    });

    it('should sort the CVSS Score column', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("CVSS Score")';
        const tdSelector = '.rt-td:nth-child(6) [data-testid="label-chip"]';

        // 0. Initial table state indicates that the column is sorted descending.
        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });

        // 1. Sort ascending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=CVSS&sort[0][desc]=false');

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 2. Sort descending by the column.
        cy.get(thSelector).click(); // no request because initial response has been cached
        cy.location('search').should('eq', '?sort[0][id]=CVSS&sort[0][desc]=true');

        cy.get(thSelector).should('have.class', '-sort-desc');
        // Do not assert because of potential timing problem: get td elements before table re-renders.
    });

    // Cluster (Platform) CVEs does not yet have Severity column.

    it('should display vulnerability descriptions', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        visitVulnerabilityManagementEntities(entitiesKey);

        // Balance positive and negative assertions.
        cy.get(selectors.cveDescription).should('exist');
        cy.get(`${selectors.cveDescription}:contains("No description available")`).should(
            'not.exist'
        );
    });

    // Argument 3 in verify functions is index of column which has the links.
    // The one-based index includes checkbox, hidden, invisible.

    it('should display links for clusters', function () {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        verifySecondaryEntities(entitiesKey, 'clusters', 9);
    });
});
