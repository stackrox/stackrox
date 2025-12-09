import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import '@testing-library/jest-dom';

import ImageComponentVulnerabilitiesTable from './ImageComponentVulnerabilitiesTable';
import type {
    ImageComponentVulnerability,
    ImageMetadataContext,
} from './ImageComponentVulnerabilitiesTable';

const mockUseFeatureFlags = vi.hoisted(() => vi.fn());

vi.mock('hooks/useFeatureFlags', () => ({
    default: mockUseFeatureFlags,
}));

// Mock table sort to avoid testing sort logic
vi.mock('hooks/useTableSort', () => ({
    default: () => ({
        sortOption: { field: 'Component', direction: 'asc' },
        getSortParams: () => ({}),
    }),
}));

describe('ImageComponentVulnerabilitiesTable', () => {
    const mockImageMetadataContext: ImageMetadataContext = {
        id: 'sha256:abc123',
        name: {
            registry: 'docker.io',
            remote: 'library/ubuntu',
            tag: '22.04',
        },
        metadata: {
            v1: {
                layers: [
                    {
                        instruction: 'FROM',
                        value: 'ubuntu:22.04',
                    },
                ],
            },
        },
    };

    // inBaseImageLayer is intentionally omitted (defaults to false/Application layer)
    const mockComponentVulnerabilities: ImageComponentVulnerability[] = [
        {
            type: 'Image',
            name: 'curl',
            version: '7.68.0-1ubuntu2',
            location: '/usr/bin/curl',
            source: 'OS',
            layerIndex: 0,
            imageVulnerabilities: [
                {
                    severity: 'CRITICAL_VULNERABILITY_SEVERITY',
                    fixedByVersion: '7.68.0-1ubuntu2.5',
                    advisory: {
                        name: 'CVE-2021-22876',
                        link: 'https://ubuntu.com/security/CVE-2021-22876',
                    },
                    pendingExceptionCount: 0,
                },
            ],
        },
    ];

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('with ROX_BASE_IMAGE_DETECTION feature flag enabled', () => {
        beforeEach(() => {
            mockUseFeatureFlags.mockReturnValue({
                isFeatureFlagEnabled: (flag: string) =>
                    flag === 'ROX_BASE_IMAGE_DETECTION' || flag === 'ROX_SCANNER_V4',
            });
        });

        it('should render Layer type column header', () => {
            render(
                <ImageComponentVulnerabilitiesTable
                    imageMetadataContext={mockImageMetadataContext}
                    componentVulnerabilities={mockComponentVulnerabilities}
                />
            );

            expect(screen.getByText('Layer type')).toBeInTheDocument();
        });

        it('should render Application label when inBaseImageLayer is false', () => {
            render(
                <ImageComponentVulnerabilitiesTable
                    imageMetadataContext={mockImageMetadataContext}
                    componentVulnerabilities={mockComponentVulnerabilities}
                />
            );

            expect(screen.getByText('Application')).toBeInTheDocument();
        });

        it('should render Base image label when inBaseImageLayer is true', () => {
            const baseImageComponent = [
                {
                    ...mockComponentVulnerabilities[0],
                    inBaseImageLayer: true,
                },
            ];

            render(
                <ImageComponentVulnerabilitiesTable
                    imageMetadataContext={mockImageMetadataContext}
                    componentVulnerabilities={baseImageComponent}
                />
            );

            expect(screen.getByText('Base image')).toBeInTheDocument();
        });
    });

    describe('with ROX_BASE_IMAGE_DETECTION feature flag disabled', () => {
        beforeEach(() => {
            mockUseFeatureFlags.mockReturnValue({
                isFeatureFlagEnabled: (flag: string) => flag === 'ROX_SCANNER_V4',
            });
        });

        it('should not render Layer type column header', () => {
            render(
                <ImageComponentVulnerabilitiesTable
                    imageMetadataContext={mockImageMetadataContext}
                    componentVulnerabilities={mockComponentVulnerabilities}
                />
            );

            expect(screen.queryByText('Layer type')).not.toBeInTheDocument();
        });
    });
});
