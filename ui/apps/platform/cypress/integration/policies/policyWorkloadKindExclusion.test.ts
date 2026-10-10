import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getInputByLabel } from '../../helpers/formHelpers';
import { deletePolicyIfExists, doPolicyPageAction, visitPolicies } from './Policies.helpers';
import { selectors } from './Policies.selectors';
import {
    addExclusionWithDeployment,
    assertStepHeading,
    assertWorkloadKindExclusion,
    clickNext,
    clickSave,
    dragFieldIntoSection,
    expandCriteriaCategory,
    goToWizardStep,
    selectCategory,
    skipFiltersStepIfPresent,
    startPolicyWizard,
    toggleWorkloadKindExclusion,
    verifyPolicyDetails,
    verifyPolicyInTable,
} from './policyWizard.helpers';

const POLICY_NAME = 'CYPRESS_TEST_WORKLOAD_KIND_EXCLUSION';
const EXCLUDED_WORKLOAD_NAME = 'system-admin';

describe('Policy workload kind exclusion', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_POLICY_WORKLOAD_KIND_EXCLUSION')) {
            this.skip();
        }
    });

    beforeEach(() => {
        deletePolicyIfExists(POLICY_NAME);
    });

    afterEach(() => {
        deletePolicyIfExists(POLICY_NAME);
    });

    it('should exclude CronJobs, persist on reopen, and combine with a workload name exclusion', () => {
        visitPolicies();
        startPolicyWizard();

        getInputByLabel('Name').type(POLICY_NAME);
        cy.contains('label', 'High').click();
        selectCategory('Privileges');
        getInputByLabel('Description').type(
            'Exclude CronJobs and a named workload from this policy'
        );
        clickNext();

        assertStepHeading('Lifecycle');
        cy.contains('label', 'Deploy').click();
        clickNext();

        assertStepHeading('Rules');
        expandCriteriaCategory('Container configuration');
        dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains("Privileged")`);
        cy.get(selectors.step3.policyCriteria.groupCards).should('have.length.gte', 1);
        clickNext();

        assertStepHeading('Resources');
        cy.contains('Exclude resources from this policy by workload kind, by scope, or both.');
        toggleWorkloadKindExclusion('CronJobs');
        assertWorkloadKindExclusion('CronJobs', true);
        assertWorkloadKindExclusion('Jobs', false);
        cy.contains('Policy will exclude CronJobs.');
        addExclusionWithDeployment(EXCLUDED_WORKLOAD_NAME);
        clickNext();

        skipFiltersStepIfPresent();

        assertStepHeading('Actions');
        clickNext();

        assertStepHeading('Review policy');
        cy.contains('dt', 'Workload kind').next('dd').should('contain', 'CronJobs');
        cy.contains(EXCLUDED_WORKLOAD_NAME);
        clickSave();

        verifyPolicyInTable(POLICY_NAME);
        verifyPolicyDetails(POLICY_NAME, {
            severity: 'High',
            lifecycle: 'Deploy',
            categories: 'Privileges',
            description: 'Exclude CronJobs and a named workload from this policy',
            response: 'Inform',
            scope: ['Workload kind', 'CronJobs', EXCLUDED_WORKLOAD_NAME],
        });

        doPolicyPageAction('Edit policy');
        goToWizardStep('Resources');
        assertWorkloadKindExclusion('CronJobs', true);
        assertWorkloadKindExclusion('Jobs', false);
        cy.get('[aria-label="Workload name"]').should('have.value', EXCLUDED_WORKLOAD_NAME);

        toggleWorkloadKindExclusion('Jobs');
        assertWorkloadKindExclusion('Jobs', true);
        cy.contains('Policy will exclude CronJobs and Jobs.');

        goToWizardStep('Review');
        cy.contains('dt', 'Workload kind').next('dd').should('have.text', 'CronJobs, Jobs');
        clickSave();

        visitPolicies();
        verifyPolicyDetails(POLICY_NAME, {
            severity: 'High',
            lifecycle: 'Deploy',
            categories: 'Privileges',
            description: 'Exclude CronJobs and a named workload from this policy',
            response: 'Inform',
            scope: ['Workload kind', 'CronJobs, Jobs', EXCLUDED_WORKLOAD_NAME],
        });
    });
});
