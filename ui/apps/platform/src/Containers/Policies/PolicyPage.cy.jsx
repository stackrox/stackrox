import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { FeatureFlagsContext } from 'hooks/useFeatureFlags';

import PolicyPage from './PolicyPage';

const mockFeatureFlags = {
    isFeatureFlagEnabled: () => false,
    isLoadingFeatureFlags: false,
    error: undefined,
};

// Generated policy with one CVSS criterion. Fields mirror the backend response, which emits
// unpopulated fields as empty arrays; getClientWizardPolicy reads exclusions without a guard, so
// omitting them would throw and surface the error modal instead of the wizard.
const generatedPolicyResponse = {
    policy: {
        id: '',
        name: '',
        description: '',
        rationale: '',
        remediation: '',
        disabled: false,
        categories: [],
        lifecycleStages: [],
        eventSource: 'NOT_APPLICABLE',
        exclusions: [],
        scope: [],
        severity: 'LOW_SEVERITY',
        enforcementActions: [],
        notifiers: [],
        mitreAttackVectors: [],
        criteriaLocked: false,
        mitreVectorsLocked: false,
        isDefault: false,
        policyVersion: '1.1',
        policySections: [
            {
                sectionName: 'Rule 1',
                policyGroups: [
                    {
                        fieldName: 'CVSS',
                        booleanOperator: 'OR',
                        negate: false,
                        values: [{ value: '>= 7' }],
                    },
                ],
            },
        ],
    },
    alteredSearchTerms: [],
    hasNestedFields: false,
};

function TestWrapper({ children }) {
    return (
        <ComponentTestProvider>
            <FeatureFlagsContext.Provider value={mockFeatureFlags}>
                {children}
            </FeatureFlagsContext.Provider>
        </ComponentTestProvider>
    );
}

function mountGeneratePage() {
    window.history.pushState({}, '', '/main/policy-management/policies?action=generate&s[CVSS]=>7');
    cy.mount(
        <TestWrapper>
            <PolicyPage hasWriteAccessForPolicy pageAction="generate" />
        </TestWrapper>
    );
}

describe(Cypress.spec.relative, () => {
    beforeEach(() => {
        cy.intercept('GET', '**/v1/policycategories*', { body: { categories: [] } });
        cy.intercept('GET', '**/v1/mitreattackvectors*', { body: { mitreAttackVectors: [] } });
    });

    it('renders a spinner while the generate request is pending and does not mount the wizard early', () => {
        cy.intercept('POST', '**/v1/policies/from-search', {
            delay: 1000,
            body: generatedPolicyResponse,
        }).as('generate');

        mountGeneratePage();

        // The wizard must not mount until the generated policy is set, otherwise Formik captures an
        // empty policy (it reads initialValues once). A spinner here, with the wizard not yet
        // rendered, is the guard.
        cy.findByRole('progressbar').should('be.visible');
        cy.findByRole('heading', { name: 'Create policy' }).should('not.exist');
    });

    it('mounts the wizard only after the generated policy resolves', () => {
        cy.intercept('POST', '**/v1/policies/from-search', {
            body: generatedPolicyResponse,
        }).as('generate');

        mountGeneratePage();
        cy.wait('@generate');

        cy.findByRole('progressbar').should('not.exist');
        cy.findByRole('heading', { name: 'Create policy' }).should('be.visible');
    });
});
