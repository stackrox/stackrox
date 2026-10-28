import { Formik } from 'formik';

import { FeatureFlagsContext } from 'hooks/useFeatureFlags';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import ReviewPolicyForm from './ReviewPolicyForm';

const defaultPolicy = {
    id: 'policy-id',
    name: 'Test policy',
    description: 'Test policy description',
    rationale: 'Test policy rationale',
    remediation: 'Test policy guidance',
    disabled: false,
    categories: ['Security Best Practices'],
    lifecycleStages: ['BUILD', 'DEPLOY'],
    eventSource: 'NOT_APPLICABLE',
    exclusions: [],
    scope: [],
    severity: 'LOW_SEVERITY',
    enforcementActions: [],
    notifiers: [],
    policyVersion: '1.1',
    policySections: [],
    mitreAttackVectors: [],
    criteriaLocked: false,
    mitreVectorsLocked: false,
    isDefault: false,
    source: 'IMPERATIVE',
    evaluationFilter: null,
    excludedDeploymentScopes: [],
    excludedImageNames: [],
};

function setup(policyOverrides = {}) {
    cy.intercept('GET', '/v1/notifiers', { notifiers: [] }).as('getNotifiers');
    cy.intercept('GET', '/v1/mitreattackvectors', { mitreAttackVectors: [] }).as(
        'getMitreAttackVectors'
    );
    cy.intercept('GET', '/v1/sac/clusters?*', { clusters: [] }).as('getClustersForPermissions');
    cy.intercept('POST', graphqlUrl('getMitreAttackVectors'), {
        data: { policy: { mitreAttackVectors: [] } },
    }).as('getPolicyMitreAttackVectors');

    cy.mount(
        <ComponentTestProvider>
            <FeatureFlagsContext.Provider
                value={{
                    isFeatureFlagEnabled: () => false,
                    isLoadingFeatureFlags: false,
                    error: undefined,
                }}
            >
                <Formik
                    initialValues={{ ...defaultPolicy, ...policyOverrides }}
                    onSubmit={() => {}}
                >
                    <ReviewPolicyForm
                        isBadRequest={false}
                        policyErrorMessage=""
                        setIsBadRequest={cy.stub().as('setIsBadRequest')}
                        setIsValidOnServer={cy.stub().as('setIsValidOnServer')}
                        setPolicyErrorMessage={cy.stub().as('setPolicyErrorMessage')}
                    />
                </Formik>
            </FeatureFlagsContext.Provider>
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('shows an inactive preview and skips dry run requests for runtime-only policies', () => {
        const startDryRun = cy.stub().as('startDryRun');
        cy.intercept('POST', '/v1/policies/dryrunjob', startDryRun);

        setup({ lifecycleStages: ['RUNTIME'] });

        cy.findByRole('heading', { name: 'Preview violations' });
        cy.findByText(
            'Runtime violations are not available in this preview because they are generated in response to future events.'
        );
        cy.findByText('Preview inactive for runtime policies');
        cy.findByText(
            'Runtime policy violations will be generated only after the policy is saved and enabled.'
        );
        cy.findByText(
            'Before you save the policy, verify that the violations seem accurate.'
        ).should('not.exist');
        cy.findByText('No deployments have Build or Deploy type violations.').should('not.exist');
        cy.wait([
            '@getNotifiers',
            '@getMitreAttackVectors',
            '@getClustersForPermissions',
            '@getPolicyMitreAttackVectors',
        ]);
        cy.get('@startDryRun').should('not.have.been.called');
    });

    it('shows dry run preview results for Build and Deploy policies', () => {
        cy.intercept('POST', '/v1/policies/dryrunjob', { jobId: 'dry-run-job-id' }).as(
            'startDryRun'
        );
        cy.intercept('GET', '/v1/policies/dryrunjob/dry-run-job-id', {
            pending: false,
            result: { alerts: [] },
        }).as('checkDryRun');

        setup();

        cy.wait('@startDryRun');
        cy.wait('@checkDryRun');
        cy.findByText(
            'The policy settings you have selected will generate violations for the Build or Deploy lifecycle stages. Runtime violations are not available in this preview because they are generated in response to future events.'
        );
        cy.findByText('Before you save the policy, verify that the violations seem accurate.');
        cy.findByText('No deployments have Build or Deploy type violations.');
        cy.findByText('Preview inactive for runtime policies').should('not.exist');
    });
});
