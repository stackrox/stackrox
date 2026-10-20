import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import type { ClientPolicy } from 'types/policy.proto';

import PolicyFiltersForm from './PolicyFiltersForm';

const mockUseFeatureFlags = vi.hoisted(() => ({
    isFeatureFlagEnabled: vi.fn(),
}));

const mockFormikContext = vi.hoisted(() => ({
    values: {} as ClientPolicy,
    setFieldValue: vi.fn(),
}));

vi.mock('hooks/useFeatureFlags', () => ({
    default: () => mockUseFeatureFlags,
}));

vi.mock('formik', () => ({
    useFormikContext: () => mockFormikContext,
}));

describe('PolicyFiltersForm', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);
        mockFormikContext.values = {
            lifecycleStages: [],
            eventSource: 'NOT_APPLICABLE',
            policySections: [],
            evaluationFilter: null,
            excludedImageNames: [],
            excludedDeploymentScopes: [],
            serverPolicySections: [],
        } as unknown as ClientPolicy;
    });

    describe('build-lifecycle info alert', () => {
        it('should render build-lifecycle alert when policy is build-only and image layer filtering is enabled', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            mockFormikContext.values = {
                lifecycleStages: ['BUILD'],
                eventSource: 'NOT_APPLICABLE',
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
                excludedImageNames: [],
                excludedDeploymentScopes: [],
                serverPolicySections: [],
            } as unknown as ClientPolicy;

            render(<PolicyFiltersForm />);

            const alert = screen.getByText(
                'Image layer filtering applies to all lifecycle stages, including Build.'
            );
            expect(alert).toBeInTheDocument();
        });

        it('should not render build-lifecycle alert when no image criteria exist', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            mockFormikContext.values = {
                lifecycleStages: ['BUILD'],
                eventSource: 'NOT_APPLICABLE',
                policySections: [],
                evaluationFilter: null,
                excludedImageNames: [],
                excludedDeploymentScopes: [],
                serverPolicySections: [],
            } as unknown as ClientPolicy;

            render(<PolicyFiltersForm />);

            expect(
                screen.queryByText(
                    'Image layer filtering applies to all lifecycle stages, including Build.'
                )
            ).not.toBeInTheDocument();
        });

        it('should not render build-lifecycle alert when policy is not build-only', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            mockFormikContext.values = {
                lifecycleStages: ['BUILD', 'DEPLOY'],
                eventSource: 'NOT_APPLICABLE',
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
                excludedImageNames: [],
                excludedDeploymentScopes: [],
                serverPolicySections: [],
            } as unknown as ClientPolicy;

            render(<PolicyFiltersForm />);

            expect(
                screen.queryByText(
                    'Image layer filtering applies to all lifecycle stages, including Build.'
                )
            ).not.toBeInTheDocument();
        });

        it('should not render build-lifecycle alert when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);

            mockFormikContext.values = {
                lifecycleStages: ['BUILD'],
                eventSource: 'NOT_APPLICABLE',
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
                excludedImageNames: [],
                excludedDeploymentScopes: [],
                serverPolicySections: [],
            } as unknown as ClientPolicy;

            render(<PolicyFiltersForm />);

            expect(
                screen.queryByText(
                    'Image layer filtering applies to all lifecycle stages, including Build.'
                )
            ).not.toBeInTheDocument();
        });

        it('should render both disabled-criteria alert and build-lifecycle alert when appropriate', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            // Build-only but no image criteria, so filtering is disabled
            mockFormikContext.values = {
                lifecycleStages: ['BUILD'],
                eventSource: 'NOT_APPLICABLE',
                policySections: [],
                evaluationFilter: null,
                excludedImageNames: [],
                excludedDeploymentScopes: [],
                serverPolicySections: [],
            } as unknown as ClientPolicy;

            render(<PolicyFiltersForm />);

            // Should show the disabled alert, not the build-lifecycle alert
            expect(
                screen.getByText(
                    'Image layer filtering requires at least one criterion from Image Contents or Image Scanning.'
                )
            ).toBeInTheDocument();
            expect(
                screen.queryByText(
                    'Image layer filtering applies to all lifecycle stages, including Build.'
                )
            ).not.toBeInTheDocument();
        });

        it('should render disabled-criteria alert when no image criteria exist and policy is not build-only', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            mockFormikContext.values = {
                lifecycleStages: ['BUILD', 'DEPLOY'],
                eventSource: 'NOT_APPLICABLE',
                policySections: [],
                evaluationFilter: null,
                excludedImageNames: [],
                excludedDeploymentScopes: [],
                serverPolicySections: [],
            } as unknown as ClientPolicy;

            render(<PolicyFiltersForm />);

            expect(
                screen.getByText(
                    'Image layer filtering requires at least one criterion from Image Contents or Image Scanning.'
                )
            ).toBeInTheDocument();
        });
    });

    describe('image layer filtering section visibility', () => {
        it('should render image layer section when ROX_IMAGE_LAYER_FILTER is ON', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockImplementation(
                (flag: string) => flag === 'ROX_IMAGE_LAYER_FILTER'
            );

            render(<PolicyFiltersForm />);

            expect(screen.getByText('Image layers')).toBeInTheDocument();
        });

        it('should not render image layer section when ROX_IMAGE_LAYER_FILTER is OFF', () => {
            mockUseFeatureFlags.isFeatureFlagEnabled.mockReturnValue(false);

            render(<PolicyFiltersForm />);

            expect(screen.queryByText('Image layers')).not.toBeInTheDocument();
        });
    });
});
