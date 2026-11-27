import { FeatureFlagsContext } from 'hooks/useFeatureFlags';

import ViolationDetails from './ViolationDetails';

const createFeatureFlagsContextValue = (featureFlagEnabled) => ({
    isFeatureFlagEnabled: (flag) => featureFlagEnabled && flag === 'ROX_IMAGE_LAYER_FILTER',
    isLoadingFeatureFlags: false,
    error: undefined,
});

describe(Cypress.spec.relative, () => {
    const defaultProps = {
        processViolation: null,
        lifecycleStage: 'DEPLOY',
        violations: [],
        evaluationFilter: null,
        lifecycleStages: ['DEPLOY'],
        eventSource: 'DEPLOYMENT_EVENT',
    };

    describe('PolicyFiltersSection rendering', () => {
        it('should render policy filters section with image layer filter when ROX_IMAGE_LAYER_FILTER is ON and skipImageLayers is SKIP_BASE', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('exist');
            cy.findByText('Application layers only').should('exist');
            cy.findByText('Image layers').should('exist');
        });

        it('should render policy filters section when ROX_IMAGE_LAYER_FILTER is ON and skipImageLayers is SKIP_APP', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_APP',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('exist');
            cy.findByText('Base layers only').should('exist');
            cy.findByText('Image layers').should('exist');
        });

        it('should not render policy filters section when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('not.exist');
            cy.findByText('Application layers only').should('not.exist');
            cy.findByText('Violation events').should('exist');
        });

        it('should not render policy filters section when no filters are active', () => {
            const evaluationFilter = {
                skipImageLayers: 'SKIP_NONE',
                skipContainerTypes: [],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('not.exist');
            cy.findByText('Violation events').should('exist');
        });

        it('should not render policy filters section when evaluationFilter is null', () => {
            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={null} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('not.exist');
            cy.findByText('Violation events').should('exist');
        });

        it('should render container type filter when skipContainerTypes includes INIT', () => {
            const evaluationFilter = {
                skipContainerTypes: ['INIT'],
                skipImageLayers: 'SKIP_NONE',
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('exist');
            cy.findByText('Skip init containers').should('exist');
        });

        it('should render both container type and image layer filters when both are set and ROX_IMAGE_LAYER_FILTER is ON', () => {
            const evaluationFilter = {
                skipContainerTypes: ['INIT'],
                skipImageLayers: 'SKIP_BASE',
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(true)}>
                    <ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Policy filters').should('exist');
            cy.findByText('Skip init containers').should('exist');
            cy.findByText('Application layers only').should('exist');
        });
    });

    describe('Violation events section', () => {
        it('should always render Violation events title', () => {
            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <ViolationDetails {...defaultProps} />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Violation events').should('exist');
        });

        it('should render DeploytimeMessages when lifecycleStage is DEPLOY', () => {
            const violations = [{ message: 'Container nginx uses an image with a disallowed tag' }];

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <ViolationDetails
                        {...defaultProps}
                        lifecycleStage="DEPLOY"
                        violations={violations}
                    />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Container nginx uses an image with a disallowed tag').should('exist');
        });

        it('should render RuntimeMessages when lifecycleStage is RUNTIME', () => {
            const processViolation = {
                message: 'Unauthorized process execution detected',
                processes: [
                    {
                        signal: {
                            id: 'process-1',
                            time: '2024-01-01T00:00:00Z',
                        },
                    },
                ],
            };

            cy.mount(
                <FeatureFlagsContext.Provider value={createFeatureFlagsContextValue(false)}>
                    <ViolationDetails
                        {...defaultProps}
                        lifecycleStage="RUNTIME"
                        processViolation={processViolation}
                    />
                </FeatureFlagsContext.Provider>
            );

            cy.findByText('Unauthorized process execution detected').should('exist');
        });
    });
});
