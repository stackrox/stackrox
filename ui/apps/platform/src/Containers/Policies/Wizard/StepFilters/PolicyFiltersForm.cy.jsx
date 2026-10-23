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

const policyWithImageCriteria = {
    ...baseClientPolicy,
    policySections: [
        {
            sectionName: 'test',
            policyGroups: [
                {
                    fieldName: 'Image Component',
                    booleanOperator: 'OR',
                    negate: false,
                    values: [],
                },
            ],
        },
    ],
    evaluationFilter: {
        skipImageLayers: 'SKIP_NONE',
        skipContainerTypes: [],
    },
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
    describe('build-lifecycle info alert', () => {
        it('should render build-lifecycle alert when policy is build-only and image layer filtering is enabled', () => {
            const policy = {
                ...policyWithImageCriteria,
                lifecycleStages: ['BUILD'],
            };

            mountWithFormik(policy, true);

            cy.findByText(
                'Image layer filtering applies to all lifecycle stages, including Build.'
            ).should('exist');
        });

        it('should not render build-lifecycle alert when no image criteria exist', () => {
            const policy = {
                ...baseClientPolicy,
                lifecycleStages: ['BUILD'],
            };

            mountWithFormik(policy, true);

            cy.findByText(
                'Image layer filtering applies to all lifecycle stages, including Build.'
            ).should('not.exist');
        });

        it('should not render build-lifecycle alert when policy is not build-only', () => {
            const policy = {
                ...policyWithImageCriteria,
                lifecycleStages: ['BUILD', 'DEPLOY'],
            };

            mountWithFormik(policy, true);

            cy.findByText(
                'Image layer filtering applies to all lifecycle stages, including Build.'
            ).should('not.exist');
        });

        it('should not render build-lifecycle alert when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            const policy = {
                ...policyWithImageCriteria,
                lifecycleStages: ['BUILD'],
            };

            mountWithFormik(policy, false);

            cy.findByText(
                'Image layer filtering applies to all lifecycle stages, including Build.'
            ).should('not.exist');
        });

        it('should render both disabled-criteria alert and build-lifecycle alert when appropriate', () => {
            const policy = {
                ...baseClientPolicy,
                lifecycleStages: ['BUILD'],
            };

            mountWithFormik(policy, true);

            cy.findByText(
                'Image layer filtering requires at least one criterion from Image Contents or Image Scanning.'
            ).should('exist');
            cy.findByText(
                'Image layer filtering applies to all lifecycle stages, including Build.'
            ).should('not.exist');
        });

        it('should render disabled-criteria alert when no image criteria exist and policy is not build-only', () => {
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
        it('should allow selecting different radio options when image criteria are present', () => {
            const policy = {
                ...policyWithImageCriteria,
                lifecycleStages: ['BUILD', 'DEPLOY'],
            };

            mountWithFormik(policy, true);

            // Initially, 'All layers' should be checked
            cy.get('#image-layer-all').should('be.checked');
            cy.get('#image-layer-app').should('not.be.checked');
            cy.get('#image-layer-base').should('not.be.checked');

            // Click 'Application layers only'
            cy.get('#image-layer-app').click();
            cy.get('#image-layer-app').should('be.checked');
            cy.get('#image-layer-all').should('not.be.checked');

            // Click 'Base layers only'
            cy.get('#image-layer-base').click();
            cy.get('#image-layer-base').should('be.checked');
            cy.get('#image-layer-app').should('not.be.checked');

            // Click back to 'All layers'
            cy.get('#image-layer-all').click();
            cy.get('#image-layer-all').should('be.checked');
            cy.get('#image-layer-base').should('not.be.checked');
        });

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
