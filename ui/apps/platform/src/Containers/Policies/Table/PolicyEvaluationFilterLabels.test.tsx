import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import type { EvaluationFilter } from 'types/policy.proto';

import PolicyEvaluationFilterLabels from './PolicyEvaluationFilterLabels';

const mockUseFeatureFlags = vi.hoisted(() => ({
    isFeatureFlagEnabled: vi.fn(),
}));

vi.mock('hooks/useFeatureFlags', () => ({
    default: () => mockUseFeatureFlags,
}));

describe('PolicyEvaluationFilterLabels', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);
    });

    describe('image layer filtering (independent of ROX_EVALUATION_FILTER)', () => {
        it('should render "App layers only" chip when SKIP_BASE and ROX_IMAGE_LAYER_FILTER is ON, even if ROX_EVALUATION_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            const chip = screen.getByText('App layers only');
            expect(chip).toBeInTheDocument();
        });

        it('should render "Base layers only" chip when SKIP_APP and ROX_IMAGE_LAYER_FILTER is ON, even if ROX_EVALUATION_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_APP',
                skipContainerTypes: [],
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            const chip = screen.getByText('Base layers only');
            expect(chip).toBeInTheDocument();
        });

        it('should not render image layer chips when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_BASE',
                skipContainerTypes: [],
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            expect(screen.queryByText('App layers only')).not.toBeInTheDocument();
        });

        it('should not render image layer chips when skipImageLayers is SKIP_NONE', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_NONE',
                skipContainerTypes: [],
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            expect(screen.queryByText('App layers only')).not.toBeInTheDocument();
            expect(screen.queryByText('Base layers only')).not.toBeInTheDocument();
        });
    });

    describe('container type filtering (gated by ROX_EVALUATION_FILTER)', () => {
        it('should render "Skips init" chip when ROX_EVALUATION_FILTER and ROX_INIT_CONTAINER_SUPPORT are ON', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation((flag: string) =>
                ['ROX_EVALUATION_FILTER', 'ROX_INIT_CONTAINER_SUPPORT'].includes(flag)
            );

            const evaluationFilter: EvaluationFilter = {
                skipContainerTypes: ['SKIP_INIT'],
                skipImageLayers: 'SKIP_NONE',
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            const chip = screen.getByText('Skips init');
            expect(chip).toBeInTheDocument();
        });

        it('should not render "Skips init" chip when ROX_EVALUATION_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_INIT_CONTAINER_SUPPORT'
            );

            const evaluationFilter: EvaluationFilter = {
                skipContainerTypes: ['SKIP_INIT'],
                skipImageLayers: 'SKIP_NONE',
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            expect(screen.queryByText('Skips init')).not.toBeInTheDocument();
        });

        it('should not render "Skips init" chip when ROX_INIT_CONTAINER_SUPPORT is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_EVALUATION_FILTER'
            );

            const evaluationFilter: EvaluationFilter = {
                skipContainerTypes: ['SKIP_INIT'],
                skipImageLayers: 'SKIP_NONE',
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            expect(screen.queryByText('Skips init')).not.toBeInTheDocument();
        });
    });

    describe('combined scenarios', () => {
        it('should render both container type and image layer chips when both features are enabled with independent flags', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(true);

            const evaluationFilter: EvaluationFilter = {
                skipContainerTypes: ['SKIP_INIT'],
                skipImageLayers: 'SKIP_BASE',
            };

            render(<PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />);

            expect(screen.getByText('Skips init')).toBeInTheDocument();
            expect(screen.getByText('App layers only')).toBeInTheDocument();
        });

        it('should return null when evaluationFilter is null', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(true);

            const { container } = render(<PolicyEvaluationFilterLabels evaluationFilter={null} />);

            expect(container).toBeEmptyDOMElement();
        });

        it('should return null when no filters are active', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(true);

            const evaluationFilter: EvaluationFilter = {
                skipImageLayers: 'SKIP_NONE',
                skipContainerTypes: [],
            };

            const { container } = render(
                <PolicyEvaluationFilterLabels evaluationFilter={evaluationFilter} />
            );

            expect(container).toBeEmptyDOMElement();
        });
    });
});
