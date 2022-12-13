import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';

import {
    getCountAndNounFromNodeCVEsLinkResults,
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from './vulnerabilityManagement.helpers';

const entitiesKey = 'node-components';

describe('Vulnerability Management Node Components', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }
    });

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasTableColumnHeadings([
            '', // hidden
            'Component',
            'Operating System',
            'Node CVEs',
            'Top CVSS',
            'Nodes',
            'Risk Priority',
        ]);
    });

    it('should sort the Risk Priority column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Risk Priority")';
        const tdSelector = '.rt-td:nth-child(7)';

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
            '?sort[0][id]=Component%20Risk%20Priority&sort[0][desc]=true'
        );

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });

        // 2. Sort ascending by the column.
        cy.get(thSelector).click(); // no request because initial response has been cached
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Component%20Risk%20Priority&sort[0][desc]=false'
        );

        cy.get(thSelector).should('have.class', '-sort-asc');
        // Do not assert because of potential timing problem: get td elements before table re-renders.
    });

    // TODO Investigate whether not yet supported or incorrect field in payload.
    it.skip('should sort the Top CVSS column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Top CVSS")';
        const tdSelector = '.rt-td:nth-child(5) [data-testid="label-chip"]';

        // 0. Initial table state indicates that the column is not sorted.
        cy.get(thSelector)
            .should('not.have.class', '-sort-asc')
            .should('not.have.class', '-sort-desc');

        // 1. Sort ascending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Component%20Top%20CVSS&sort[0][desc]=false'
        );

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
        });

        // 2. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should(
            'eq',
            '?sort[0][id]=Component%20Top%20CVSS&sort[0][desc]=true'
        );

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
        });
    });

    // Argument 3 in verify functions is index of column which has the links.
    // The one-based index includes checkbox, hidden, invisible.

    // Some tests might fail in local deployment.

    it('should display links for all node CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'node-cves',
            4,
            /^\d+ CVEs?$/,
            getCountAndNounFromNodeCVEsLinkResults
        );
    });

    it('should display links for fixable node CVEs', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'node-cves',
            4,
            /^\d+ Fixable$/,
            getCountAndNounFromNodeCVEsLinkResults
        );
    });

    it('should display links for nodes', () => {
        verifySecondaryEntities(entitiesKey, 'nodes', 6, /^\d+ nodes?$/);
    });
});
