import React, { ReactElement, ReactNode } from 'react';
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
import { useParams } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';
import isEmpty from 'lodash/isEmpty';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import useURLPagination from 'hooks/useURLPagination';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';
import { getOverviewPagePath } from '../../utils/searchUtils';
import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import useHasGenerateSBOMAbility from '../../hooks/useHasGenerateSBOMAbility';
import ImagePageVulnerabilities from './ImagePageVulnerabilities';
import ImagePageResources from './ImagePageResources';
import { detailsTabValues } from '../../types';
import ImageDetailBadges, {
    ImageDetails,
    imageDetailsFragment,
} from '../components/ImageDetailBadges';
import getImageScanMessage from '../utils/getImageScanMessage';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { getImageBaseNameDisplay } from '../utils/images';

const workloadCveOverviewImagePath = getOverviewPagePath('Workload', {
    vulnerabilityState: 'OBSERVED',
    entityTab: 'Image',
});

export const imageDetailsQuery = gql`
    ${imageDetailsFragment}
    query getImageDetails($id: ID!) {
        image(id: $id) {
            id
            name {
                registry
                remote
                tag
            }
            ...ImageDetails
        }
    }
`;

function ScannerV4RequiredTooltip({ children }: { children: ReactElement }) {
    return (
        <Tooltip content="SBOM generation requires Scanner V4 to be enabled">{children}</Tooltip>
    );
}

function ImagePage() {
    const { imageId } = useParams();
    const { data, error } = useQuery<
        {
            image: {
                id: string;
                name: {
                    registry: string;
                    remote: string;
                    tag: string;
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

    const hasGenerateSBOMAbility = useHasGenerateSBOMAbility();
    const isScannerV4Enabled = useIsScannerV4Enabled();

    const imageData = data && data.image;
    const imageName = imageData?.name;
    const imageDisplayName =
        imageData && imageName
            ? `${imageName.registry}/${getImageBaseNameDisplay(imageData.id, imageName)}`
            : 'NAME UNKNOWN';
    const scanMessage = getImageScanMessage(imageData?.notes || [], imageData?.scanNotes || []);

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
        const SBOMButtonWrapper = isScannerV4Enabled ? React.Fragment : ScannerV4RequiredTooltip;
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
                                {hasGenerateSBOMAbility && (
                                    <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                                        <SBOMButtonWrapper>
                                            <Button
                                                variant="secondary"
                                                onClick={() => {}}
                                                isAriaDisabled={!isScannerV4Enabled}
                                            >
                                                Generate SBOM
                                            </Button>
                                        </SBOMButtonWrapper>
                                    </FlexItem>
                                )}
                            </Flex>
                            {!isEmpty(scanMessage) && (
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
                        onSelect={(e, key) => {
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
                            />
                        </Tab>
                        <Tab
                            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                            eventKey="Resources"
                            title={<TabTitleText>Resources</TabTitleText>}
                        >
                            <ImagePageResources imageId={imageId} pagination={pagination} />
                        </Tab>
                    </Tabs>
                </PageSection>
            </>
        );
    }

    return (
        <>
            <PageTitle title={`Workload CVEs - Image ${imageData ? imageDisplayName : ''}`} />
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
        </>
    );
}

export default ImagePage;
