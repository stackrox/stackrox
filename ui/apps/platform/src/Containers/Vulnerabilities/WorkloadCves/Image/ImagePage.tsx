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
import GenerateSbomModal, {
    getSbomGenerationStatusMessage,
} from '../../components/GenerateSbomModal';
import ScanHistoryModal from '../../components/ScanHistoryModal';
import { getOverviewPagePath } from '../../utils/searchUtils';
import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import useHasGenerateSbomAbility from '../../hooks/useHasGenerateSBOMAbility';
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
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { getImagesScanHistory } from 'services/imageService';

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

function ImagePage() {
    const { imageId } = useParams();
    const { getAbsoluteUrl, pageTitle } = useWorkloadCveViewContext();
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

    const hasGenerateSbomAbility = useHasGenerateSbomAbility();
    const isScannerV4Enabled = useIsScannerV4Enabled();
    const [sbomTargetImage, setSbomTargetImage] = useState<string>();
    const [historyTargetImage, setHistoryTargetImage] = useState<string>();

    const imageData = data && data.image;
    const imageName = imageData?.name;
    const imageDisplayName =
        imageData && imageName
            ? `${imageName.registry}/${getImageBaseNameDisplay(imageData.id, imageName)}`
            : 'NAME UNKNOWN';
    const scanMessage = getImageScanMessage(imageData?.notes || [], imageData?.scanNotes || []);
    const hasScanMessage = !isEmpty(scanMessage);

    const workloadCveOverviewImagePath = getAbsoluteUrl(
        getOverviewPagePath('Workload', {
            vulnerabilityState: 'OBSERVED',
            entityTab: 'Image',
        })
    );

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
                                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                                        <Button
                                            variant="secondary"
                                            onClick={() => {
                                                setHistoryTargetImage(imageData.id);
                                                console.log(`Dave: ${imageData.id}`)
                                            }}
                                        >
                                            History
                                        </Button>
                                        {historyTargetImage && (
                                            <ScanHistoryModal
                                                onClose={() => setHistoryTargetImage(undefined)}
                                                imageName={historyTargetImage}
                                            />
                                        )}
                                    </FlexItem>
                                {hasGenerateSbomAbility && (
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
        </>
    );
}

export default ImagePage;
