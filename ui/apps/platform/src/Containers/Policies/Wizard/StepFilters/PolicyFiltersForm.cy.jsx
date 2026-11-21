import { Formik } from 'formik';

import { FeatureFlagsContext } from 'hooks/useFeatureFlags';

import PolicyFiltersForm from './PolicyFiltersForm';

const createFeatureFlagsContextValue = (featureFlagEnabled) => ({
    isFeatureFlagEnabled: (flag) => featureFlagEnabled && flag === 'ROX_IMAGE_LAYER_FILTER',
    isLoadingFeatureFlags: false,
    error: undefined,
});

const baseClientPolicy = {
    lifecycleStages: [],
    eventSource: 'NOT_APPLICABLE',
    policySections: [],
    evaluationFilter: null,
    excludedImageNames: [],
    excludedDeploymentScopes: [],
    serverPolicySections: [],
};

function mountWithFormik(policy, flagEnabled) {
    cy.mount(
        <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(flagEnabled)}>
            <Formik initialValues={policy} onSubmit={() => {}}>
                <PolicyFiltersForm />
            </Formik>
        </FeatureFlagsContext.Provider>
    );
}

describe(Cypress.spec.relative, () => {
    describe('disabled-criteria info alert', () => {
        it('should render disabled-criteria alert when no image criteria exist', () => {
            const policy = {
                ...baseClientPolicy,
                lifecycleStages: ['BUILD', 'DEPLOY'],
            };

            mountWithFormik(policy, true);

            cy.findByText(
                'Image layer filtering requires at least one criterion from Image Contents or Image Scanning.'
            ).should('exist');
        });
    });

    describe('image layer filtering section visibility', () => {
        it('should render image layer section when ROX_IMAGE_LAYER_FILTER is ON', () => {
            mountWithFormik(baseClientPolicy, true);

            cy.findByText('Image layers').should('exist');
        });

        it('should not render image layer section when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            mountWithFormik(baseClientPolicy, false);

            cy.findByText('Image layers').should('not.exist');
        });
    });

    describe('image layer radio interactions', () => {
        it('should disable radio buttons when no image criteria exist', () => {
            const policy = {
                ...baseClientPolicy,
                lifecycleStages: ['BUILD', 'DEPLOY'],
            };

            mountWithFormik(policy, true);

            cy.get('#image-layer-all').should('be.disabled');
            cy.get('#image-layer-app').should('be.disabled');
            cy.get('#image-layer-base').should('be.disabled');
        });
    });
});
