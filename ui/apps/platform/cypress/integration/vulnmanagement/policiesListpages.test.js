import { selectors } from '../../constants/VulnManagementPage';
import { selectors as policySelectors } from '../../constants/PoliciesPagePatternFly';
import withAuth from '../../helpers/basicAuth';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    verifyFilteredSecondaryEntitiesLink,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

export function getPanelHeaderTextFromLinkResults([, count]) {
    return {
        panelHeaderText: `${count} ${count === 1 ? 'deployment' : 'deployments'}`,
    };
}

const entitiesKey = 'policies';

describe('Vulnerability Management Policies', () => {
    withAuth();

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns(
            [
                'Policy',
                'Description',
                'Policy Status',
                'Last Updated',
                'Latest Violation',
                'Severity',
                'Deployments',
                'Lifecycle',
                'Enforcement',
            ],
            2 // skip 2 additional columns to account for checkbox column, and untitled Statuses column
        );
    });

    it('should display links for failing deployments', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'deployments',
            4,
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
