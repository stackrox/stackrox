import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    interactAndWaitForVulnerabilityManagementEntities,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

// After the problem has been fixed, remove the function argument below.
function getCountAndNounFromClustersLinkResults(resultsFromRegExp) {
    const relatedEntitiesCount = resultsFromRegExp[1];
    const relatedEntitiesNoun = relatedEntitiesCount === 1 ? 'CLUSTER' : 'CLUSTERS';
    return {
        panelHeaderText: '0 clusters', // workaround for bug
        relatedEntitiesCount,
        relatedEntitiesNoun,
    };
}

const entitiesKey = 'cluster-cves';

describe('Vulnerability Management Cluster (Platform) CVEs', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }
    });

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns(
            [
                'CVE',
                'Type',
                'Fixable',
                'CVSS Score',
                'Env. Impact',
                'Impact Score',
                'Entities',
                'Published',
            ],
            1 // skip 1 additional column to account for checkbox column
        );
    });

    it('should sort the CVSS Score column', () => {
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

    it('should display vulnerability descriptions', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        // Balance positive and negative assertions.
        cy.get(selectors.cveDescription).should('exist');
        cy.get(`${selectors.cveDescription}:contains("No description available")`).should(
            'not.exist'
        );
    });

    // Argument 3 in verify functions is one-based index of column which has the links.
    // Count the checkbox as the first column.

    // Some tests might fail in local deployment.

    // TODO Investigate why CI displays No clusters instead of 2 clusters.
    it.skip('should display links for clusters', () => {
        verifySecondaryEntities(
            entitiesKey,
            'clusters',
            8,
            /^\d+ clusters?$/,
            getCountAndNounFromClustersLinkResults
        );
    });
});
