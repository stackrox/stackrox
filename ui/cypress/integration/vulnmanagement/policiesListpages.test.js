import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';
import { url, selectors } from '../../constants/VulnManagementPage';
import { selectors as policySelectors } from '../../constants/PoliciesPage';
import { hasExpectedHeaderColumns, allChecksForEntities } from '../../helpers/vmWorkflowUtils';

describe('Policies list Page and its entity detail page , related entities sub list  validations ', () => {
    before(function beforeHook() {
        // skip the whole suite if vuln mgmt isn't enabled
        if (checkFeatureFlag('ROX_VULN_MGMT_UI', false)) {
            this.skip();
        }
    });

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
            if (columnValue !== 'no deployments' && columnValue.includes('deployment'))
                allChecksForEntities(url.list.policies, 'deployment');
        });
    });

    describe('pre-Boolean Policy Logic tests (deprecated)', () => {
        before(function beforeHook() {
            // skip the whole suite if BPL is enabled
            if (checkFeatureFlag('ROX_BOOLEAN_POLICY_LOGIC', true)) {
                this.skip();
            }
        });

        // regression test for ROX-4752
        it('should show Privileged criterion when present in the policy', () => {
            cy.visit(url.list.policies);

            cy.get(`${selectors.tableRows}:contains('Privileged')`).click();

            cy.get('[data-testid="widget-body"] [data-testid="privileged"]')
                .invoke('text')
                .then((criterionText) => {
                    expect(criterionText).to.contain('Yes');
                });
        });
    });

    describe('post-Boolean Policy Logic tests', () => {
        before(function beforeHook() {
            // skip the whole suite if BPL is not enabled
            if (checkFeatureFlag('ROX_BOOLEAN_POLICY_LOGIC', false)) {
                this.skip();
            }
        });

        // regression test for ROX-4752
        it('should show Privileged criterion when present in the policy', () => {
            cy.visit(url.list.policies);

            cy.get(`${selectors.tableRows}:contains('Privileged')`).click();

            cy.get(
                `${policySelectors.booleanPolicySection.policyFieldCard}:contains("Privileged Container Status") ${policySelectors.booleanPolicySection.policyFieldValue}:first button`
            ).should('have.value', 'true');
        });
    });
});
