import { FeatureFlagsContext } from 'hooks/useFeatureFlags';

import PolicyEvaluationFilterLabels from './PolicyEvaluationFilterLabels';

const createFeatureFlagsContextValue = (featureFlagEnabled) => ({
    isFeatureFlagEnabled: (flag) => featureFlagEnabled && flag === 'ROX_IMAGE_LAYER_FILTER',
    isLoadingFeatureFlags: false,
    error: undefined,
});

describe(Cypress.spec.relative, () => {
    describe('image layer filtering', () => {
        it('should render "App layers only" chip when SKIP_BASE and ROX_IMAGE_LAYER_FILTER is ON', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('App layers only').should('exist');
        });

        it('should render "Base layers only" chip when SKIP_APP and ROX_IMAGE_LAYER_FILTER is ON', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_APP',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Base layers only').should('exist');
        });

        it('should not render image layer chips when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('App layers only').should('not.exist');
        });

        it('should not render image layer chips when skipImageLayers is SKIP_NONE', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_NONE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('App layers only').should('not.exist');
            cy.findByText('Base layers only').should('not.exist');
        });
    });

    describe('container type filtering', () => {
        it('should render "Skips init" chip when skipContainerTypes includes INIT', () => {
            const evaluationFilter = {
                skipContainerTypes: ['INIT'],
                skipImageLayers: 'SKIP_NONE',
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Skips init').should('exist');
        });

        it('should not render "Skips init" chip when skipContainerTypes is empty', () => {
            const evaluationFilter = {
                skipContainerTypes: [],
                skipImageLayers: 'SKIP_NONE',
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Skips init').should('not.exist');
        });
    });

    describe('combined scenarios', () => {
        it('should render both container type and image layer chips when both are set and ROX_IMAGE_LAYER_FILTER is ON', () => {
            const evaluationFilter = {
                skipContainerTypes: ['INIT'],
                skipImageLayers: 'SKIP_BASE',
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Skips init').should('exist');
            cy.findByText('App layers only').should('exist');
        });

        it('should return null when evaluationFilter is null', () => {
            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={null} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('App layers only').should('not.exist');
            cy.findByText('Base layers only').should('not.exist');
            cy.findByText('Skips init').should('not.exist');
        });

        it('should return null when no filters are active', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_NONE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('App layers only').should('not.exist');
            cy.findByText('Base layers only').should('not.exist');
            cy.findByText('Skips init').should('not.exist');
        });
    });
});
