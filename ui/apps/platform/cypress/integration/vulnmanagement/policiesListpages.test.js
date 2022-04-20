import withAuth from '../../helpers/basicAuth';
import { url, selectors } from '../../constants/VulnManagementPage';
import { selectors as policySelectors } from '../../constants/PoliciesPagePatternFly';
import { hasExpectedHeaderColumns, allChecksForEntities } from '../../helpers/vmWorkflowUtils';

describe('Policies list Page and its entity detail page , related entities sub list  validations ', () => {
    withAuth();

    it('should display all the columns and links expected in clusters list page', () => {
        cy.visit(url.list.policies);
        hasExpectedHeaderColumns([
            'Policy',
            'Description',
            'Policy Status',
            'Last Updated',
            'Latest Violation',
            'Severity',
            'Deployments',
            // 'Lifecycle',
            'Enforcement',
        ]);
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
            cy.visit(url.list.policies);

            cy.get(`${selectors.tableRows}:contains('Privileged')`).click();

            cy.get(
                `${policySelectors.step3.policyCriteria.groupCards}:contains("Privileged container status") ${policySelectors.step3.policyCriteria.value.radioGroupItem}:first button`
            ).should('have.class', 'pf-m-selected');
        });
    });
});
