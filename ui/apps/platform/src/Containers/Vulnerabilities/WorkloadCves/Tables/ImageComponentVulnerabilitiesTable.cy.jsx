import { FeatureFlagsContext } from 'hooks/useFeatureFlags';
import ImageComponentVulnerabilitiesTable from './ImageComponentVulnerabilitiesTable';

// Mock provider that enables the required feature flags
const MockFeatureFlagsProvider = ({ children }) => (
    <FeatureFlagsContext.Provider
        value={{
            isFeatureFlagEnabled: (flag) =>
                flag === 'ROX_BASE_IMAGE_DETECTION' || flag === 'ROX_SCANNER_V4',
            isLoadingFeatureFlags: false,
            error: undefined,
        }}
    >
        {children}
    </FeatureFlagsContext.Provider>
);

const mockImageMetadataContext = {
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

const createMockComponent = (overrides = {}) => ({
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
    ...overrides,
});

describe('ImageComponentVulnerabilitiesTable', () => {
    describe('Layer type column', () => {
        it('should render Layer type column header', () => {
            const components = [createMockComponent()];

            cy.mount(
                <MockFeatureFlagsProvider>
                    <ImageComponentVulnerabilitiesTable
                        imageMetadataContext={mockImageMetadataContext}
                        componentVulnerabilities={components}
                    />
                </MockFeatureFlagsProvider>
            );

            cy.contains('th', 'Layer type').should('be.visible');
        });

        it('should render Application label when inBaseImageLayer is false or undefined', () => {
            const components = [createMockComponent({ inBaseImageLayer: false })];

            cy.mount(
                <MockFeatureFlagsProvider>
                    <ImageComponentVulnerabilitiesTable
                        imageMetadataContext={mockImageMetadataContext}
                        componentVulnerabilities={components}
                    />
                </MockFeatureFlagsProvider>
            );

            cy.contains('Application').should('be.visible');
        });

        it('should render Base image label when inBaseImageLayer is true', () => {
            const components = [createMockComponent({ inBaseImageLayer: true })];

            cy.mount(
                <MockFeatureFlagsProvider>
                    <ImageComponentVulnerabilitiesTable
                        imageMetadataContext={mockImageMetadataContext}
                        componentVulnerabilities={components}
                    />
                </MockFeatureFlagsProvider>
            );

            cy.contains('Base image').should('be.visible');
        });

        it('should render mixed layer types correctly', () => {
            const components = [
                createMockComponent({ name: 'curl', inBaseImageLayer: true }),
                createMockComponent({ name: 'nginx', inBaseImageLayer: false }),
            ];

            cy.mount(
                <MockFeatureFlagsProvider>
                    <ImageComponentVulnerabilitiesTable
                        imageMetadataContext={mockImageMetadataContext}
                        componentVulnerabilities={components}
                    />
                </MockFeatureFlagsProvider>
            );

            cy.contains('Base image').should('be.visible');
            cy.contains('Application').should('be.visible');
        });
    });
});
