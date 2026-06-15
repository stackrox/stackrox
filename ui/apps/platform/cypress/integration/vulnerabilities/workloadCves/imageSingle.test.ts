import withAuth from '../../../helpers/basicAuth';
import { compoundFiltersSelectors } from '../../../helpers/compoundFilters';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
    interceptAndOverrideFeatureFlags,
    interceptAndOverridePermissions,
} from '../../../helpers/request';
import { verifyColumnManagement } from '../../../helpers/tableHelpers';

import { selectors as vulnSelectors } from '../vulnerabilities.selectors';
import {
    clickFirstImageWithMockedResponses,
    interactAndWaitForImageList,
    mockSbomGenerationRequest,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Image Single page', () => {
    withAuth();

    function visitFirstImage(): Cypress.Chainable<string> {
        visitWorkloadCveOverview();

        interactAndWaitForImageList(() => {
            selectEntityTab('Image');
        });

        // Ensure the data in the table has settled
        cy.get(selectors.isUpdatingTable).should('not.exist');

        return clickFirstImageWithMockedResponses();
    }

    it('should contain the correct search filters in the toolbar', () => {
        visitFirstImage();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(compoundFiltersSelectors.entityMenuToggle).click();
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Image');
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Image component');
        cy.get(compoundFiltersSelectors.entityMenuToggle).click();
    });

    // Verifies that the data returned by the server is not duplicated due to Apollo client cache issues
    // see: https://issues.redhat.com/browse/ROX-24254
    //      https://github.com/stackrox/stackrox/pull/6156
    it('should display nested component data correctly when processed via apollo client', () => {
        const isFlattenImageData = hasFeatureFlag('ROX_FLATTEN_IMAGE_DATA');
        const imageRootKey = isFlattenImageData ? 'imageV2' : 'image';

        const opname = 'getCVEsForImage';

        const imageData = {
            id: isFlattenImageData
                ? '4c657931-d333-5cb8-8f0d-7e3836525ec7'
                : 'sha256:010fec71f42f4b5e65f3f56f10af94a7c05c9c271a9bbc3026684ba170698cb5',
            ...(isFlattenImageData
                ? {
                      digest: 'sha256:010fec71f42f4b5e65f3f56f10af94a7c05c9c271a9bbc3026684ba170698cb5',
                  }
                : {}),
            name: {
                registry: 'quay.io',
                remote: 'openshift-release-dev/ocp-v4.0-art-dev',
                tag: '',
                __typename: 'ImageName',
            },
            metadata: {
                v1: {
                    layers: [
                        {
                            instruction: 'ADD',
                            value: 'file:091e888311e2628528312ffc60e27702fe04b23f8e4c95b456c16a967cdd89e0 in /',
                            __typename: 'ImageLayer',
                        },
                    ],
                    __typename: 'V1Metadata',
                },
                __typename: 'ImageMetadata',
            },
            __typename: isFlattenImageData ? 'ImageV2' : 'Image',
            imageCVECountBySeverity: {
                unknown: {
                    total: 0,
                    fixable: 0,
                    __typename: 'ResourceCountByFixability',
                },
                low: {
                    total: 0,
                    fixable: 0,
                    __typename: 'ResourceCountByFixability',
                },
                moderate: {
                    total: 1,
                    fixable: 1,
                    __typename: 'ResourceCountByFixability',
                },
                important: {
                    total: 0,
                    fixable: 0,
                    __typename: 'ResourceCountByFixability',
                },
                critical: {
                    total: 0,
                    fixable: 0,
                    __typename: 'ResourceCountByFixability',
                },
                __typename: 'ResourceCountByCVESeverity',
            },
            imageVulnerabilities: [
                {
                    severity: 'MODERATE_VULNERABILITY_SEVERITY',
                    cve: '[CYPRESS-MOCKED] CVE-2023-44487',
                    summary: 'HTTP/2 Stream Cancellation Attack',
                    cvss: 5.300000190734863,
                    scoreVersion: 'V3',
                    discoveredAtImage: '2024-04-03T19:44:55.837891332Z',
                    pendingExceptionCount: 0,
                    imageComponents: [
                        {
                            name: 'golang.org/x/net',
                            version: 'v0.13.0',
                            location: 'usr/bin/cluster-samples-operator-watch',
                            source: 'GO',
                            layerIndex: 0,
                            imageVulnerabilities: [
                                {
                                    vulnerabilityId: 'CVE-2023-44487#rhel:9',
                                    severity: 'MODERATE_VULNERABILITY_SEVERITY',
                                    fixedByVersion: '0.17.0',
                                    pendingExceptionCount: 0,
                                    __typename: 'ImageVulnerability',
                                },
                            ],
                            __typename: 'ImageComponent',
                        },
                        {
                            name: 'google.golang.org/grpc',
                            version: 'v1.54.0',
                            location: 'usr/bin/cluster-samples-operator-watch',
                            source: 'GO',
                            layerIndex: 0,
                            imageVulnerabilities: [
                                {
                                    vulnerabilityId: 'CVE-2023-44487#rhel:9',
                                    severity: 'MODERATE_VULNERABILITY_SEVERITY',
                                    fixedByVersion: '1.56.3',
                                    pendingExceptionCount: 0,
                                    __typename: 'ImageVulnerability',
                                },
                            ],
                            __typename: 'ImageComponent',
                        },
                        {
                            name: 'openshift4/ose-cluster-samples-rhel9-operator',
                            version: 'v4.15.0-202401261531.p0.gd546ec2.assembly.stream',
                            location:
                                'root/buildinfo/Dockerfile-openshift-ose-cluster-samples-rhel9-operator-v4.15.0-202401261531.p0.gd546ec2.assembly.stream',
                            source: 'OS',
                            layerIndex: 0,
                            imageVulnerabilities: [
                                {
                                    vulnerabilityId: 'CVE-2023-44487#rhel:9',
                                    severity: 'MODERATE_VULNERABILITY_SEVERITY',
                                    fixedByVersion:
                                        'v4.15.0-202404031310.p0.gbf845b5.assembly.stream.el9',
                                    pendingExceptionCount: 0,
                                    __typename: 'ImageVulnerability',
                                },
                            ],
                            __typename: 'ImageComponent',
                        },
                    ],
                    __typename: 'ImageVulnerability',
                },
            ],
        };

        const body = {
            data: {
                [imageRootKey]: imageData,
            },
        };

        // Navigate to the image page manually instead of using visitFirstImage(),
        // so we can provide our own getCVEsForImage response without alias conflicts.
        visitWorkloadCveOverview();
        interactAndWaitForImageList(() => {
            selectEntityTab('Image');
        });

        const detailRouteMatcher = getRouteMatcherMapForGraphQL(['getImageDetails', opname]);
        const detailStaticResponse = {
            getImageDetails: {
                fixture: 'vulnerabilities/workloadCves/imageWithMultipleCves.json',
            },
            [opname]: { body },
        };

        interactAndWaitForResponses(
            () => {
                cy.get('tbody tr td[data-label="Image"] a').first().click();
            },
            detailRouteMatcher,
            detailStaticResponse
        );

        cy.get(vulnSelectors.expandRowButton).click();

        const fixedInCellSelector = `table td[data-label="CVE fixed in"]`;
        const components = imageData.imageVulnerabilities[0].imageComponents;

        components.forEach((component, index) => {
            cy.get(fixedInCellSelector)
                .eq(index)
                .contains(component.imageVulnerabilities[0].fixedByVersion);
        });
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the CVE table', () => {
            visitFirstImage();
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });

    describe('SBOM generation tests', () => {
        const headerSbomModalButton = 'section:has(h1) button:contains("Generate SBOM")';
        const generateSbomButton = '[role="dialog"] button:contains("Generate SBOM")';

        it('should hide the SBOM generation button when the user does not have write access to the Image resource', () => {
            interceptAndOverridePermissions({ Image: 'READ_ACCESS' });

            visitFirstImage();

            cy.get(headerSbomModalButton).should('not.exist');
        });

        it('should disable the SBOM generation button when Scanner V4 is not enabled', () => {
            interceptAndOverrideFeatureFlags({ ROX_SCANNER_V4: false });

            visitFirstImage();

            cy.get(headerSbomModalButton).should('have.attr', 'aria-disabled', 'true');
        });

        it('should trigger a download of the image SBOM via confirmation modal', function () {
            if (!hasFeatureFlag('ROX_SCANNER_V4')) {
                this.skip();
            }

            mockSbomGenerationRequest();

            visitFirstImage().then((imageFullName) => {
                cy.get(headerSbomModalButton).click();
                cy.get(selectors.generateSbomModal).contains(imageFullName);
                cy.get(generateSbomButton).click();
                cy.get(':contains("Software Bill of Materials (SBOM) generated successfully")');
            });
        });
    });
});
