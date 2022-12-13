import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
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
} from './vulnerabilityManagement.helpers';
import { selectors } from './vulnerabilityManagement.selectors';

const entitiesKey = 'cves';

describe('Vulnerability Management CVEs', () => {
    withAuth();

    before(function beforeHook() {
        if (hasFeatureFlag('ROX_POSTGRES_DATASTORE')) {
            this.skip();
        }
    });

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
            'Discovered Time',
            'Published',
            '', // hidden
        ]);
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

    it('should display vulnerability descriptions', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        // Balance positive and negative assertions.
        cy.get(selectors.cveDescription).should('exist');
        cy.get(`${selectors.cveDescription}:contains("No description available")`).should(
            'not.exist'
        );
    });

    // Argument 3 in verify functions is index of column which has the links.
    // The one-based index includes checkbox, hidden, invisible.

    // Some tests might fail in local deployment.

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 9, /^\d+ deployments?$/);
    });

    it('should display links for images', () => {
        verifySecondaryEntities(entitiesKey, 'images', 9, /^\d+ images?$/);
    });

    it('should display links for components', () => {
        verifySecondaryEntities(entitiesKey, 'components', 9, /^\d+ components?$/);
    });
});
