import React, { ReactElement, ReactNode, useState } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
    ClipboardCopy,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Skeleton,
    Tab,
    Tabs,
    TabTitleText,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { useParams } from 'react-router-dom-v5-compat';
import { gql, useQuery } from '@apollo/client';
import isEmpty from 'lodash/isEmpty';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import usePermissions from 'hooks/usePermissions';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import type { VulnerabilityState } from 'types/cve.proto';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';
import GenerateSbomModal, {
    getSbomGenerationStatusMessage,
} from '../../components/GenerateSbomModal';
import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import ImagePageVulnerabilities from './ImagePageVulnerabilities';
import ImagePageResources from './ImagePageResources';
import ImagePageSignatureVerification from './ImagePageSignatureVerification';
import { detailsTabValues } from '../../types';
import ImageDetailBadges, {
    ImageDetails,
    imageDetailsFragment,
} from '../components/ImageDetailBadges';
import getImageScanMessage from '../utils/getImageScanMessage';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { getImageBaseNameDisplay } from '../utils/images';
import { parseQuerySearchFilter, getVulnStateScopedQueryString } from '../../utils/searchUtils';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { defaultColumns as deploymentResourcesDefaultColumns } from './DeploymentResourceTable';
import CreateReportDropdown from '../components/CreateReportDropdown';
import CreateViewBasedReportModal from '../components/CreateViewBasedReportModal';

export const imageDetailsQuery = gql`
    ${imageDetailsFragment}
    query getImageDetails($id: ID!) {
        image(id: $id) {
            id
            name {
                registry
                remote
                tag
                fullName
            }
            ...ImageDetails
        }
    }
`;

function OptionalSbomButtonTooltip({
    children,
    message,
}: {
    children: ReactElement;
    message?: string;
}) {
    if (!message) {
        return children;
    }
    return <Tooltip content={message}>{children}</Tooltip>;
}

export type ImagePageProps = {
    vulnerabilityState: VulnerabilityState;
    showVulnerabilityStateTabs: boolean;
    deploymentResourceColumnOverrides: ColumnConfigOverrides<
        keyof typeof deploymentResourcesDefaultColumns
    >;
};

function ImagePage({
    vulnerabilityState,
    showVulnerabilityStateTabs,
    deploymentResourceColumnOverrides,
}: ImagePageProps) {
    const { urlBuilder, pageTitle, baseSearchFilter, viewContext } = useWorkloadCveViewContext();
    const { imageId } = useParams() as { imageId: string };
    const { data, error } = useQuery<
        {
            image: {
                id: string;
                name: {
                    registry: string;
                    remote: string;
                    tag: string;
                    fullName: string;
                } | null;
            } & ImageDetails;
        },
        {
            id: string;
        }
    >(imageDetailsQuery, {
        variables: { id: imageId },
    });
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);
    const { invalidateAll: refetchAll } = useInvalidateVulnerabilityQueries();

    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    // Search filter management
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForImage = hasReadWriteAccess('Image'); // SBOM Generation mutates image scan state.
    const isScannerV4Enabled = useIsScannerV4Enabled();
    const [sbomTargetImage, setSbomTargetImage] = useState<string>();

    // Report-specific functionality
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isViewBasedReportsEnabled = isFeatureFlagEnabled('ROX_VULNERABILITY_VIEW_BASED_REPORTS');
    const [isCreateViewBasedReportModalOpen, setIsCreateViewBasedReportModalOpen] = useState(false);

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const onReportSelect = (_value: string | number | undefined) => {
        setIsCreateViewBasedReportModalOpen(true);
    };

    const buildImageQuery = () => {
        // Create a scoped query that includes the image SHA filter plus any applied search filters
        const imageScopedFilter = { 'Image SHA': [imageId] };
        const combinedFilter = { ...baseSearchFilter, ...imageScopedFilter, ...querySearchFilter };
        return getVulnStateScopedQueryString(combinedFilter, vulnerabilityState);
    };

    const imageData = data && data.image;
    const imageName = imageData?.name;
    const imageDisplayName =
        imageData && imageName
            ? `${imageName.registry}/${getImageBaseNameDisplay(imageData.id, imageName)}`
            : 'NAME UNKNOWN';
    const scanMessage = getImageScanMessage(imageData?.notes || [], imageData?.scanNotes || []);
    const hasScanMessage = !isEmpty(scanMessage);

    const workloadCveOverviewImagePath = urlBuilder.imageList('OBSERVED');

    let mainContent: ReactNode | null = null;

    if (error) {
        mainContent = (
            <PageSection variant="light">
                <Bullseye>
                    <EmptyStateTemplate
                        title={getAxiosErrorMessage(error)}
                        headingLevel="h2"
                        icon={ExclamationCircleIcon}
                        iconClassName="pf-v5-u-danger-color-100"
                    />
                </Bullseye>
            </PageSection>
        );
    } else {
        const sha = imageData?.id;
        mainContent = (
            <>
                <PageSection variant="light">
                    {imageData ? (
                        <Flex
                            direction={{ default: 'column' }}
                            alignItems={{ default: 'alignItemsStretch' }}
                        >
                            <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }}>
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                >
                                    <Title headingLevel="h1">{imageDisplayName}</Title>
                                    {sha && (
                                        <ClipboardCopy
                                            hoverTip="Copy SHA"
                                            clickTip="Copied!"
                                            variant="inline-compact"
                                            className="pf-v5-u-font-size-sm"
                                        >
                                            {sha}
                                        </ClipboardCopy>
                                    )}
                                    <ImageDetailBadges imageData={imageData} />
                                </Flex>
                                {hasWriteAccessForImage && (
                                    <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                                        <OptionalSbomButtonTooltip
                                            message={getSbomGenerationStatusMessage({
                                                isScannerV4Enabled,
                                                hasScanMessage,
                                            })}
                                        >
                                            <Button
                                                variant="secondary"
                                                onClick={() => {
                                                    setSbomTargetImage(imageData.name?.fullName);
                                                }}
                                                isAriaDisabled={
                                                    !isScannerV4Enabled || hasScanMessage
                                                }
                                            >
                                                Generate SBOM
                                            </Button>
                                        </OptionalSbomButtonTooltip>
                                        {sbomTargetImage && (
                                            <GenerateSbomModal
                                                onClose={() => setSbomTargetImage(undefined)}
                                                imageName={sbomTargetImage}
                                            />
                                        )}
                                    </FlexItem>
                                )}
                            </Flex>
                            {hasScanMessage && (
                                <Alert
                                    className="pf-v5-u-w-100"
                                    variant="warning"
                                    isInline
                                    title="CVE data may be inaccurate"
                                    component="p"
                                >
                                    <Flex
                                        direction={{ default: 'column' }}
                                        spaceItems={{ default: 'spaceItemsSm' }}
                                    >
                                        <FlexItem>{scanMessage.header}</FlexItem>
                                        <FlexItem>{scanMessage.body}</FlexItem>
                                    </Flex>
                                </Alert>
                            )}
                        </Flex>
                    ) : (
                        <HeaderLoadingSkeleton
                            nameScreenreaderText="Loading image name"
                            metadataScreenreaderText="Loading image metadata"
                        />
                    )}
                </PageSection>
                <PageSection
                    className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                    padding={{ default: 'noPadding' }}
                >
                    <Tabs
                        activeKey={activeTabKey}
                        onSelect={(_e, key) => {
                            setActiveTabKey(key);
                            pagination.setPage(1);
                        }}
                        className="pf-v5-u-pl-md pf-v5-u-background-color-100"
                        mountOnEnter
                        unmountOnExit
                    >
                        <Tab
                            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                            eventKey="Vulnerabilities"
                            title={<TabTitleText>Vulnerabilities</TabTitleText>}
                        >
                            <ImagePageVulnerabilities
                                imageId={imageId}
                                imageName={
                                    imageData?.name ?? {
                                        registry: '',
                                        remote: '',
                                        tag: '',
                                    }
                                }
                                refetchAll={refetchAll}
                                pagination={pagination}
                                vulnerabilityState={vulnerabilityState}
                                showVulnerabilityStateTabs={showVulnerabilityStateTabs}
                                searchFilter={searchFilter}
                                setSearchFilter={setSearchFilter}
                                additionalToolbarItems={
                                    isViewBasedReportsEnabled && (
                                        <CreateReportDropdown onSelect={onReportSelect} />
                                    )
                                }
                            />
                        </Tab>
                        <Tab
                            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                            eventKey="Resources"
                            title={<TabTitleText>Resources</TabTitleText>}
                        >
                            <ImagePageResources
                                imageId={imageId}
                                pagination={pagination}
                                deploymentResourceColumnOverrides={
                                    deploymentResourceColumnOverrides
                                }
                            />
                        </Tab>
                        <Tab
                            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                            eventKey="Signature verification"
                            title={<TabTitleText>Signature verification</TabTitleText>}
                        >
                            <ImagePageSignatureVerification
                                results={imageData?.signatureVerificationData?.results}
                            />
                        </Tab>
                    </Tabs>
                </PageSection>
            </>
        );
    }

    return (
        <>
            <PageTitle title={`${pageTitle} - Image ${imageData ? imageDisplayName : ''}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewImagePath}>
                        Images
                    </BreadcrumbItemLink>
                    {!error && (
                        <BreadcrumbItem isActive>
                            {imageData ? (
                                imageDisplayName
                            ) : (
                                <Skeleton screenreaderText="Loading image name" width="200px" />
                            )}
                        </BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            {mainContent}
            {isViewBasedReportsEnabled && (
                <CreateViewBasedReportModal
                    isOpen={isCreateViewBasedReportModalOpen}
                    setIsOpen={setIsCreateViewBasedReportModalOpen}
                    query={buildImageQuery()}
                    areaOfConcern={viewContext}
                />
            )}
        </>
    );
}

export default ImagePage;
