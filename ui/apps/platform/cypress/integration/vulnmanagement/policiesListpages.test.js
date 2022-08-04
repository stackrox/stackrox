import { url, selectors } from '../../constants/VulnManagementPage';
import { selectors as policySelectors } from '../../constants/PoliciesPagePatternFly';
import withAuth from '../../helpers/basicAuth';
import { hasExpectedHeaderColumns, allChecksForEntities } from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Policies list Page and its entity detail page , related entities sub list  validations ', () => {
    withAuth();

    it('should display all the columns and links expected in clusters list page', () => {
        visitVulnerabilityManagementEntities('policies');
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
        cy.get(selectors.tableBodyColumn).each(($el) => {
            const columnValue = $el.text().toLowerCase();
            if (
                columnValue !== 'no failing deployments' &&
                columnValue.includes('failing deployments')
            ) {
                allChecksForEntities(url.list.policies, 'deployment');
            }
        });
    });

    describe('post-Boolean Policy Logic tests', () => {
        // regression test for ROX-4752
        it('should show Privileged criterion when present in the policy', () => {
            visitVulnerabilityManagementEntities('policies');

            cy.get(`${selectors.tableRows}:contains('Privileged')`).click();

            cy.get(
                `${policySelectors.step3.policyCriteria.groupCards}:contains("Privileged container status") ${policySelectors.step3.policyCriteria.value.radioGroupItem}:first button`
            ).should('have.class', 'pf-m-selected');
        });
    });
});
