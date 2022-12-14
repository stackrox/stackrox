import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';

import {
    getCountAndNounFromImageCVEsLinkResults,
    getCountAndNounFromNodeCVEsLinkResults,
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';

function getCountAndNounFromClusterCVEsLinkResults([, count]) {
    return {
        panelHeaderText: `${count} Platform ${count === '1' ? 'CVE' : 'CVES'}`,
        relatedEntitiesCount: count,
        relatedEntitiesNoun: count === '1' ? 'CLUSTER CVE' : 'CLUSTER CVES',
    };
}

const entitiesKey = 'clusters';

describe('Vulnerability Management Clusters', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }
    });

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
            'Policy Status',
            'Latest Violation',
            'Risk Priority',
        ]);
    });

    it('should sort the Risk Priority column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Risk Priority")';
        const tdSelector = '.rt-td:nth-child(10)';

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

    // Some tests might fail in local deployment.

    it('should display links for all image CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'image-cves',
            3,
            /^\d+ CVEs?$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for fixable image CVEs', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'image-cves',
            3,
            /^\d+ Fixable$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for all node CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'node-cves',
            4,
            /^\d+ CVEs?$/,
            getCountAndNounFromNodeCVEsLinkResults
        );
    });

    it('should display links for all cluster CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'cluster-cves',
            5,
            /^\d+ CVEs?$/,
            getCountAndNounFromClusterCVEsLinkResults
        );
    });

    it('should display links for namespaces', () => {
        verifySecondaryEntities(entitiesKey, 'namespaces', 7, /^\d+ namespaces?$/);
    });

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 7, /^\d+ deployments?$/);
    });

    it('should display links for nodes', () => {
        verifySecondaryEntities(entitiesKey, 'nodes', 7, /^\d+ nodes?$/);
    });
});
