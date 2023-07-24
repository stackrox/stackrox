import withAuth from '../../helpers/basicAuth';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import {
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    verifyConditionalCVEs,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';

const entitiesKey = 'clusters';

describe('Vulnerability Management Clusters', () => {
    withAuth();

    it('should display all the columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasTableColumnHeadings([
            '', // hidden
            'Cluster',
            'Image CVEs',
            'Node CVEs',
            'Platform CVEs',
            'K8S Version',
            'Entities',
            'Latest Violation',
            'Risk Priority',
        ]);
    });

    it('should sort the Risk Priority column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Risk Priority")';
        const tdSelector = '.rt-td:nth-child(9)';

        // 0. Initial table state indicates that the column is sorted ascending.
        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 1. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Cluster%20Risk%20Priority&sort[0][desc]=true'
        );

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });

        // 2. Sort ascending by the column.
        cy.get(thSelector).click(); // no request because initial response has been cached
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Cluster%20Risk%20Priority&sort[0][desc]=false'
        );

        cy.get(thSelector).should('have.class', '-sort-asc');
        // Do not assert because of potential timing problem: get td elements before table re-renders.
    });

    // Argument 3 in verify functions is index of column which has the links.
    // The one-based index includes checkbox, hidden, invisible.

    it('should display either links for image CVEs or text for No CVEs', () => {
        verifyConditionalCVEs(entitiesKey, 'image-cves', 3, 'imageVulnerabilityCounter');
    });

    it('should display either links for node CVEs or text for No CVEs', () => {
        verifyConditionalCVEs(entitiesKey, 'node-cves', 4, 'nodeVulnerabilityCounter');
    });

    it('should display either links for cluster CVEs or text for No CVEs', () => {
        verifyConditionalCVEs(entitiesKey, 'cluster-cves', 5, 'clusterVulnerabilityCounter');
    });

    it('should display links for namespaces', () => {
        verifySecondaryEntities(entitiesKey, 'namespaces', 7);
    });

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 7);
    });

    it('should display links for nodes', () => {
        verifySecondaryEntities(entitiesKey, 'nodes', 7);
    });
});
