import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import type { EvaluationFilter, LifecycleStage, PolicyEventSource } from 'types/policy.proto';
import type { ProcessViolation, Violation } from 'types/alert.proto';

import ViolationDetails from './ViolationDetails';

const mockUseFeatureFlags = vi.hoisted(() => ({
    isFeatureFlagEnabled: vi.fn(),
}));

vi.mock('hooks/useFeatureFlags', () => ({
    default: () => mockUseFeatureFlags,
}));

// Mock child components to simplify testing
vi.mock('./DeploytimeMessages', () => ({
    default: () => <div>Deploytime Messages</div>,
}));

vi.mock('./RuntimeMessages', () => ({
    default: () => <div>Runtime Messages</div>,
}));

describe('ViolationDetails', () => {
    const defaultProps = {
        processViolation: null as ProcessViolation | null,
        lifecycleStage: 'DEPLOY' as LifecycleStage,
        violations: [] as Violation[],
        evaluationFilter: null as EvaluationFilter | null,
        lifecycleStages: ['DEPLOY'] as LifecycleStage[],
        eventSource: 'DEPLOYMENT_EVENT' as PolicyEventSource,
    };

    beforeEach(() => {
        vi.clearAllMocks();
        mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);
    });

    describe('PolicyFiltersSection rendering', () => {
        it('should render policy filters section with image layer filter when ROX_IMAGE_LAYER_FILTER is ON and skipImageLayers is SKIP_BASE', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            render(<ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />);

            // PolicyFiltersSection should render with "Policy filters" heading
            expect(screen.getByText('Policy filters')).toBeInTheDocument();
            // And should show "Application layers only" label
            expect(screen.getByText('Application layers only')).toBeInTheDocument();
            // And should show "Image layers" term
            expect(screen.getByText('Image layers')).toBeInTheDocument();
        });

        it('should render policy filters section when ROX_IMAGE_LAYER_FILTER is ON and skipImageLayers is SKIP_APP', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_APP',
                skipContainerTypes: [],
            };

            render(<ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />);

            expect(screen.getByText('Policy filters')).toBeInTheDocument();
            expect(screen.getByText('Base layers only')).toBeInTheDocument();
            expect(screen.getByText('Image layers')).toBeInTheDocument();
        });

        it('should not render policy filters section when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            render(<ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />);

            // PolicyFiltersSection should return null and not render
            expect(screen.queryByText('Policy filters')).not.toBeInTheDocument();
            expect(screen.queryByText('Application layers only')).not.toBeInTheDocument();
            // ViolationDetails should still render
            expect(screen.getByText('Violation events')).toBeInTheDocument();
        });

        it('should not render policy filters section when no filters are active', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(true);

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_NONE',
                skipContainerTypes: [],
            };

            render(<ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />);

            // PolicyFiltersSection should return null when no filters apply
            expect(screen.queryByText('Policy filters')).not.toBeInTheDocument();
            // ViolationDetails should still render
            expect(screen.getByText('Violation events')).toBeInTheDocument();
        });

        it('should not render policy filters section when evaluationFilter is null', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(true);

            render(<ViolationDetails {...defaultProps} evaluationFilter={null} />);

            expect(screen.queryByText('Policy filters')).not.toBeInTheDocument();
            // ViolationDetails should still render
            expect(screen.getByText('Violation events')).toBeInTheDocument();
        });

        it('should render container type filter when ROX_EVALUATION_FILTER and ROX_INIT_CONTAINER_SUPPORT are ON', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation((flag: string) =>
                ['ROX_EVALUATION_FILTER', 'ROX_INIT_CONTAINER_SUPPORT'].includes(flag)
            );

            const evaluationFilter: EvaluationFilter = {
                skipContainerTypes: ['INIT'],
                skipImageLayers: 'SKIP_NONE',
            };

            render(<ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />);

            expect(screen.getByText('Policy filters')).toBeInTheDocument();
            expect(screen.getByText('Skip init containers')).toBeInTheDocument();
        });

        it('should render both container type and image layer filters when both features are enabled', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(true);

            const evaluationFilter: EvaluationFilter = {
                skipContainerTypes: ['INIT'],
                skipImageLayers: 'SKIP_BASE',
            };

            render(<ViolationDetails {...defaultProps} evaluationFilter={evaluationFilter} />);

            expect(screen.getByText('Policy filters')).toBeInTheDocument();
            expect(screen.getByText('Skip init containers')).toBeInTheDocument();
            expect(screen.getByText('Application layers only')).toBeInTheDocument();
        });
    });

    describe('Violation events section', () => {
        it('should always render Violation events title', () => {
            render(<ViolationDetails {...defaultProps} />);

            expect(screen.getByText('Violation events')).toBeInTheDocument();
        });

        it('should render DeploytimeMessages when lifecycleStage is DEPLOY', () => {
            render(<ViolationDetails {...defaultProps} lifecycleStage="DEPLOY" />);

            expect(screen.getByText('Deploytime Messages')).toBeInTheDocument();
        });

        it('should render RuntimeMessages when lifecycleStage is RUNTIME', () => {
            render(<ViolationDetails {...defaultProps} lifecycleStage="RUNTIME" />);

            expect(screen.getByText('Runtime Messages')).toBeInTheDocument();
        });
    });
});
