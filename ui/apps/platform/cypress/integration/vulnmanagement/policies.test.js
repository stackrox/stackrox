import { selectors as policySelectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import {
    assertSortedItems,
    callbackForPairOfAscendingPolicySeverityValuesFromElements,
    callbackForPairOfDescendingPolicySeverityValuesFromElements,
} from '../../helpers/sort';

import {
    hasTableColumnHeadings,
    interactAndWaitForVulnerabilityManagementEntities,
    verifyFilteredSecondaryEntitiesLink,
    visitVulnerabilityManagementEntities,
} from './VulnerabilityManagement.helpers';
import { selectors } from './VulnerabilityManagement.selectors';

export function getPanelHeaderTextFromLinkResults([, count]) {
    return {
        panelHeaderText: `${count} ${count === '1' ? 'deployment' : 'deployments'}`,
    };
}

const entitiesKey = 'policies';

describe('Vulnerability Management Policies', () => {
    withAuth();

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasTableColumnHeadings([
            '', // checkbox
            '', // hidden
            '', // invisible
            'Policy',
            'Description',
            'Policy Status',
            'Last Updated',
            'Latest Violation',
            'Severity',
            'Deployments',
            'Lifecycle',
            'Enforcement',
            '', // hidden
        ]);
    });

    it('should sort the Severity column', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        const thSelector = '.rt-th:contains("Severity")';
        const tdSelector = '.rt-td:nth-child(9)';

        // 0. Initial table state indicates that the column is not sorted.
        cy.get(thSelector)
            .should('not.have.class', '-sort-asc')
            .should('not.have.class', '-sort-desc');

        // 1. Sort ascending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Severity&sort[0][desc]=false');

        cy.get(thSelector).should('have.class', '-sort-asc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfAscendingPolicySeverityValuesFromElements);
        });

        // 2. Sort descending by the column.
        interactAndWaitForVulnerabilityManagementEntities(() => {
            cy.get(thSelector).click();
        }, entitiesKey);
        cy.location('search').should('eq', '?sort[0][id]=Severity&sort[0][desc]=true');

        cy.get(thSelector).should('have.class', '-sort-desc');
        cy.get(tdSelector).then((items) => {
            assertSortedItems(items, callbackForPairOfDescendingPolicySeverityValuesFromElements);
        });
    });

    // Argument 3 in verify functions is index of column which has the links.
    // The one-based index includes checkbox, hidden, invisible.

    // Some tests might fail in local deployment.

    it('should display links for failing deployments', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'deployments',
            10,
            /^\d+ failing deployments?$/,
            getPanelHeaderTextFromLinkResults
        );
    });

    describe('post-Boolean Policy Logic tests', () => {
        // regression test for ROX-4752
        it('should show Privileged criterion when present in the policy', () => {
            visitVulnerabilityManagementEntities('policies');

            // Pulling policy "Fixable CVSS >= 6 and Privileged" and avoiding "Privileged Container(s) with Important and Critical CVE(s)"
            cy.get(`${selectors.tableRows}:contains('and Privileged')`).click();

            cy.get(
                `${policySelectors.step3.policyCriteria.groupCards}:contains("Privileged container status") ${policySelectors.step3.policyCriteria.value.radioGroupItem}:first button`
            ).should('have.class', 'pf-m-selected');
        });
    });
});
