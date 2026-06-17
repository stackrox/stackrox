import { selectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getInputByLabel } from '../../helpers/formHelpers';
import { deletePolicyIfExists, visitPolicies } from '../../helpers/policies';
import {
    addExclusionWithDeployment,
    addInclusionWithNamespace,
    assertCriteriaCategories,
    assertStepHeading,
    clickNext,
    clickSave,
    dragFieldIntoSection,
    enableEnforcement,
    expandCriteriaCategory,
    selectCategory,
    startPolicyWizard,
    verifyPolicyDetails,
    verifyPolicyInTable,
} from '../../helpers/policyWizardHelpers';

// Exact criteria categories for each policy type
const CRITERIA_CATEGORIES = {
    build: ['Image registry', 'Image contents', 'Image scanning'],
    deploy: [
        'Image registry',
        'Image contents',
        'Image scanning',
        'Container configuration',
        'Deployment metadata',
        'Storage',
        'Networking',
        'Access control',
    ],
    runtimeDeployment: [
        'Image registry',
        'Image contents',
        'Image scanning',
        'Container configuration',
        'Deployment metadata',
        'Storage',
        'Networking',
        'Access control',
        'Process activity',
        'Baseline deviation',
        'User issued container commands',
        'File activity',
    ],
    runtimeAuditLog: ['Resource operation (Required)', 'Resource attributes'],
    runtimeNode: ['File activity', 'Process activity'],
};

const POLICY_NAMES = {
    build: 'CYPRESS_TEST_BUILD_POLICY',
    deploy: 'CYPRESS_TEST_DEPLOY_POLICY',
    runtimeDeployment: 'CYPRESS_TEST_RUNTIME_DEPLOY_POLICY',
    runtimeAuditLog: 'CYPRESS_TEST_RUNTIME_AUDIT_POLICY',
    runtimeNode: 'CYPRESS_TEST_RUNTIME_NODE_POLICY',
};

describe('Policy creation workflow', () => {
    withAuth();

    beforeEach(() => {
        Object.values(POLICY_NAMES).forEach(deletePolicyIfExists);
    });

    afterEach(() => {
        Object.values(POLICY_NAMES).forEach(deletePolicyIfExists);
    });

    it('should create a Build policy with NVD CVSS criteria and build enforcement', () => {
        const policyName = POLICY_NAMES.build;
        visitPolicies();
        startPolicyWizard();

        // Step 1 — Details
        getInputByLabel('Name').type(policyName);
        cy.contains('label', 'Critical').click();
        selectCategory('Vulnerability Management');
        getInputByLabel('Description').type('Build policy for image scanning');
        clickNext();

        // Step 2 — Lifecycle
        assertStepHeading('Lifecycle');
        cy.contains('label', 'Build').click();
        clickNext();

        // Step 3 — Rules
        assertStepHeading('Rules');
        assertCriteriaCategories(CRITERIA_CATEGORIES.build);
        expandCriteriaCategory('Image scanning');
        dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains("NVD CVSS")`);

        cy.get(selectors.step3.policyCriteria.value.select).click();
        cy.get(
            `${selectors.step3.policyCriteria.value.selectOption}:contains("Is greater than or equal to")`
        ).click();
        cy.get(selectors.step3.policyCriteria.value.numberInput).clear();
        cy.get(selectors.step3.policyCriteria.value.numberInput).type('7');

        cy.get(selectors.step3.policyCriteria.groupCards).should('have.length.gte', 1);
        clickNext();

        // Step 4 — Resources
        assertStepHeading('Resources');
        clickNext();

        // Filters
        if (
            hasFeatureFlag('ROX_EVALUATION_FILTER') &&
            hasFeatureFlag('ROX_INIT_CONTAINER_SUPPORT')
        ) {
            assertStepHeading('Filters');
            clickNext();
        }

        // Step 5 — Actions
        assertStepHeading('Actions');
        enableEnforcement('Enforce on Build');
        clickNext();

        // Step 6 — Review
        assertStepHeading('Review policy');
        clickSave();
        verifyPolicyInTable(policyName);
        verifyPolicyDetails(policyName, {
            severity: 'Critical',
            lifecycle: 'Build',
            categories: 'Vulnerability Management',
            description: 'Build policy for image scanning',
            response: 'Enforce',
            enforcement: 'Build',
            criteria: ['NVD CVSS', 'Is greater than or equal to', '7'],
        });
    });

    it('should create a Deploy policy with Privileged criteria, scoping, and deploy enforcement', () => {
        const policyName = POLICY_NAMES.deploy;
        visitPolicies();
        startPolicyWizard();

        // Step 1 — Details
        getInputByLabel('Name').type(policyName);
        cy.contains('label', 'High').click();
        selectCategory('Privileges');
        getInputByLabel('Description').type('Deploy policy checking container privileges');
        getInputByLabel('Rationale').type('Privileged containers have full host access');
        getInputByLabel('Guidance').type('Remove privileged flag from container spec');
        clickNext();

        // Step 2 — Lifecycle
        assertStepHeading('Lifecycle');
        cy.contains('label', 'Deploy').click();
        clickNext();

        // Step 3 — Rules
        assertStepHeading('Rules');
        assertCriteriaCategories(CRITERIA_CATEGORIES.deploy);
        expandCriteriaCategory('Container configuration');
        dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains("Privileged")`);
        cy.get(selectors.step3.policyCriteria.groupCards).should('have.length.gte', 1);
        clickNext();

        // Step 4 — Resources
        assertStepHeading('Resources');
        addInclusionWithNamespace('test-namespace');
        addExclusionWithDeployment('system-admin');
        clickNext();

        // Filters (container type filter enabled for Deploy lifecycle)
        const hasFiltersStep =
            hasFeatureFlag('ROX_EVALUATION_FILTER') && hasFeatureFlag('ROX_INIT_CONTAINER_SUPPORT');
        if (hasFiltersStep) {
            assertStepHeading('Filters');
            cy.contains('label', 'Skip init containers').click();
            clickNext();
        }

        // Step 5 — Actions
        assertStepHeading('Actions');
        enableEnforcement('Enforce on Deploy');
        clickNext();

        // Step 6 — Review
        assertStepHeading('Review policy');
        clickSave();
        verifyPolicyInTable(policyName);
        verifyPolicyDetails(policyName, {
            severity: 'High',
            lifecycle: 'Deploy',
            categories: 'Privileges',
            description: 'Deploy policy checking container privileges',
            response: 'Enforce',
            enforcement: 'Deploy',
            criteria: ['Privileged'],
            scope: ['test-namespace', 'system-admin'],
            ...(hasFiltersStep ? { filters: { containerTypes: 'Skip init containers' } } : {}),
        });
    });

    it('should create a Runtime policy with Process name criteria and inform-only response', () => {
        const policyName = POLICY_NAMES.runtimeDeployment;
        visitPolicies();
        startPolicyWizard();

        // Step 1 — Details
        getInputByLabel('Name').type(policyName);
        cy.contains('label', 'Medium').click();
        selectCategory('Anomalous Activity');
        getInputByLabel('Description').type('Runtime policy monitoring process execution');
        clickNext();

        // Step 2 — Lifecycle
        assertStepHeading('Lifecycle');
        cy.contains('label', 'Runtime').click();
        cy.contains('label', 'Deployment').click();
        clickNext();

        // Step 3 — Rules
        assertStepHeading('Rules');
        assertCriteriaCategories(CRITERIA_CATEGORIES.runtimeDeployment);
        expandCriteriaCategory('Process activity');
        dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains("Process name")`);
        cy.get(selectors.step3.policyCriteria.value.textInput).type('apt-get');
        cy.get(selectors.step3.policyCriteria.groupCards).should('have.length.gte', 1);
        clickNext();

        // Step 4 — Resources
        assertStepHeading('Resources');
        addInclusionWithNamespace('default');
        clickNext();

        // Filters
        if (
            hasFeatureFlag('ROX_EVALUATION_FILTER') &&
            hasFeatureFlag('ROX_INIT_CONTAINER_SUPPORT')
        ) {
            assertStepHeading('Filters');
            clickNext();
        }

        // Step 5 — Actions
        assertStepHeading('Actions');
        cy.findByLabelText('Inform').should('be.checked');
        clickNext();

        // Step 6 — Review
        assertStepHeading('Review policy');
        clickSave();
        verifyPolicyInTable(policyName);
        verifyPolicyDetails(policyName, {
            severity: 'Medium',
            lifecycle: 'Runtime',
            categories: 'Anomalous Activity',
            description: 'Runtime policy monitoring process execution',
            response: 'Inform',
        });
    });

    it('should create a Runtime Audit Log policy with Kubernetes API verb criteria', () => {
        const policyName = POLICY_NAMES.runtimeAuditLog;
        visitPolicies();
        startPolicyWizard();

        // Step 1 — Details
        getInputByLabel('Name').type(policyName);
        cy.contains('label', 'Critical').click();
        selectCategory('Kubernetes Events');
        clickNext();

        // Step 2 — Lifecycle
        assertStepHeading('Lifecycle');
        cy.contains('label', 'Runtime').click();
        cy.contains('label', 'Audit logs').click();
        clickNext();

        // Step 3 — Rules (audit log requires both API verb and resource type)
        assertStepHeading('Rules');
        assertCriteriaCategories(CRITERIA_CATEGORIES.runtimeAuditLog);
        expandCriteriaCategory('Resource operation (Required)');
        dragFieldIntoSection(
            `${selectors.step3.policyCriteria.key}:contains("Kubernetes API verb")`
        );

        cy.get(selectors.step3.policyCriteria.value.select).click();
        cy.get(`${selectors.step3.policyCriteria.value.selectOption}:contains("CREATE")`).click();

        dragFieldIntoSection(
            `${selectors.step3.policyCriteria.key}:contains("Kubernetes resource type")`
        );
        cy.get(selectors.step3.policyCriteria.value.select).eq(1).click();
        cy.get(`${selectors.step3.policyCriteria.value.selectOption}:contains("Secrets")`).click();

        cy.get(selectors.step3.policyCriteria.groupCards).should('have.length.gte', 2);
        clickNext();

        // Step 4 — Resources (audit log disables image exclusions but allows namespace scoping)
        assertStepHeading('Resources');
        clickNext();

        // Filters
        if (
            hasFeatureFlag('ROX_EVALUATION_FILTER') &&
            hasFeatureFlag('ROX_INIT_CONTAINER_SUPPORT')
        ) {
            assertStepHeading('Filters');
            clickNext();
        }

        // Step 5 — Actions (enforcement disabled for audit log)
        assertStepHeading('Actions');
        cy.findByLabelText('Inform and enforce').should('be.disabled');
        clickNext();

        // Step 6 — Review
        assertStepHeading('Review policy');
        clickSave();
        verifyPolicyInTable(policyName);
        verifyPolicyDetails(policyName, {
            severity: 'Critical',
            lifecycle: 'Runtime',
            categories: 'Kubernetes Events',
            response: 'Inform',
        });
    });

    it('should create a Runtime Node Event policy with File activity criteria', function () {
        if (!hasFeatureFlag('ROX_SENSITIVE_FILE_ACTIVITY')) {
            this.skip();
        }

        const policyName = POLICY_NAMES.runtimeNode;
        visitPolicies();
        startPolicyWizard();

        // Step 1 — Details
        getInputByLabel('Name').type(policyName);
        cy.contains('label', 'High').click();
        selectCategory('System Modification');
        clickNext();

        // Step 2 — Lifecycle
        assertStepHeading('Lifecycle');
        cy.contains('label', 'Runtime').click();
        cy.contains('label', 'Node').click();
        clickNext();

        // Step 3 — Rules
        assertStepHeading('Rules');
        assertCriteriaCategories(CRITERIA_CATEGORIES.runtimeNode);
        expandCriteriaCategory('File activity');
        dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains("File path")`);
        cy.get(selectors.step3.policyCriteria.value.textInput).type('/etc/passwd');

        dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains("File operation")`);
        cy.get(selectors.step3.policyCriteria.value.select).click();
        cy.get(
            `${selectors.step3.policyCriteria.value.selectOption}:contains("Permission change")`
        ).click();

        cy.get(selectors.step3.policyCriteria.groupCards).should('have.length.gte', 2);
        clickNext();

        // Step 4 — Resources (scoping disabled for node events)
        assertStepHeading('Resources');
        cy.contains('The selected event source does not support resource targeting.');
        cy.contains('button', 'Add inclusion').should('be.disabled');
        cy.contains('button', 'Add exclusion').should('be.disabled');
        clickNext();

        // Filters
        if (
            hasFeatureFlag('ROX_EVALUATION_FILTER') &&
            hasFeatureFlag('ROX_INIT_CONTAINER_SUPPORT')
        ) {
            assertStepHeading('Filters');
            clickNext();
        }

        // Step 5 — Actions (enforcement disabled for node events)
        assertStepHeading('Actions');
        cy.findByLabelText('Inform and enforce').should('be.disabled');
        clickNext();

        // Step 6 — Review
        assertStepHeading('Review policy');
        clickSave();
        verifyPolicyInTable(policyName);
        verifyPolicyDetails(policyName, {
            severity: 'High',
            lifecycle: 'Runtime',
            categories: 'System Modification',
            response: 'Inform',
        });
    });
});
