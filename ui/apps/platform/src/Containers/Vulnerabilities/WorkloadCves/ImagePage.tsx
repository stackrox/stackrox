import React, { ReactNode, useState } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    Flex,
    Label,
    LabelGroup,
    PageSection,
    Skeleton,
    Tab,
    Tabs,
    TabsComponent,
    TabTitleText,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { CopyIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { getDateTime, getDistanceStrictAsPhrase } from 'utils/dateUtils';
import useURLStringUnion from 'hooks/useURLStringUnion';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ImagePageVulnerabilities from './ImagePageVulnerabilities';
import ImagePageResources from './ImagePageResources';
import { detailsTabValues } from './types';
import { getOverviewCvesPath } from './searchUtils';
import useImageDetails, { ImageDetailsResponse } from './hooks/useImageDetails';

const workloadCveOverviewImagePath = getOverviewCvesPath({
    cveStatusTab: 'Observed',
    entityTab: 'Image',
});

function ImageDetailBadges({ imageData }: { imageData: ImageDetailsResponse['image'] }) {
    const [hasSuccessfulCopy, setHasSuccessfulCopy] = useState(false);

    const { deploymentCount, operatingSystem, metadata, dataSource, scanTime } = imageData;
    const created = metadata?.v1?.created;
    const sha = metadata?.v1?.digest;
    const isActive = deploymentCount > 0;

    function copyToClipboard(imageSha: string) {
        navigator.clipboard
            .writeText(imageSha)
            .then(() => setHasSuccessfulCopy(true))
            .catch(() => {
                // Permission is not required to write to the clipboard in secure contexts when initiated
                // via a user event so this Promise should not reject
            })
            .finally(() => {
                setTimeout(() => setHasSuccessfulCopy(false), 2000);
            });
    }

    return (
        <LabelGroup numLabels={Infinity}>
            <Label isCompact color={isActive ? 'green' : 'gold'}>
                {isActive ? 'Active' : 'Inactive'}
            </Label>
            {operatingSystem && <Label isCompact>OS: {operatingSystem}</Label>}
            {created && (
                <Label isCompact>Age: {getDistanceStrictAsPhrase(created, new Date())}</Label>
            )}
            {scanTime && (
                <Label isCompact>
                    Scan time: {getDateTime(scanTime)} by {dataSource?.name ?? 'Unknown Scanner'}
                </Label>
            )}
            {sha && (
                <Tooltip content="Copy image SHA to clipboard">
                    <Label
                        style={{ cursor: 'pointer' }}
                        icon={<CopyIcon />}
                        isCompact
                        color={hasSuccessfulCopy ? 'green' : 'grey'}
                        onClick={() => copyToClipboard(sha)}
                    >
                        {hasSuccessfulCopy ? 'Copied!' : 'SHA'}
                    </Label>
                </Tooltip>
            )}
        </LabelGroup>
    );
}

function ImagePage() {
    const { imageId } = useParams();
    const { data, error } = useImageDetails(imageId);
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const imageData = data && data.image;
    const imageName = imageData?.name?.fullName ?? 'NAME UNKNOWN';

    let mainContent: ReactNode | null = null;

    if (error) {
        mainContent = (
            <PageSection variant="light">
                <Bullseye>
                    <EmptyStateTemplate
                        title={getAxiosErrorMessage(error)}
                        headingLevel="h2"
                        icon={ExclamationCircleIcon}
                        iconClassName="pf-u-danger-color-100"
                    />
                </Bullseye>
            </PageSection>
        );
    } else {
        mainContent = (
            <>
                <PageSection variant="light">
                    {imageData ? (
                        <Flex direction={{ default: 'column' }}>
                            <Title headingLevel="h1" className="pf-u-mb-sm">
                                {imageName}
                            </Title>
                            <ImageDetailBadges imageData={imageData} />
                        </Flex>
                    ) : (
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsXs' }}
                            className="pf-u-w-50"
                        >
                            <Skeleton screenreaderText="Loading image name" fontSize="2xl" />
                            <Skeleton screenreaderText="Loading image metadata" fontSize="sm" />
                        </Flex>
                    )}
                </PageSection>
                <PageSection
                    className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                    padding={{ default: 'noPadding' }}
                >
                    <Tabs
                        activeKey={activeTabKey}
                        onSelect={(e, key) => setActiveTabKey(key)}
                        component={TabsComponent.nav}
                        className="pf-u-pl-md pf-u-background-color-100"
                        mountOnEnter
                        unmountOnExit
                    >
                        <Tab
                            className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                            eventKey="Vulnerabilities"
                            title={<TabTitleText>Vulnerabilities</TabTitleText>}
                        >
                            <ImagePageVulnerabilities imageId={imageId} />
                        </Tab>
                        <Tab
                            className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                            eventKey="Resources"
                            title={<TabTitleText>Resources</TabTitleText>}
                            isDisabled
                        >
                            <ImagePageResources />
                        </Tab>
                    </Tabs>
                </PageSection>
            </>
        );
    }

    return (
        <>
            <PageTitle title={`Workload CVEs - Image ${imageData ? imageName : ''}`} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewImagePath}>
                        Images
                    </BreadcrumbItemLink>
                    {!error && (
                        <BreadcrumbItem isActive>
                            {imageData ? (
                                imageName
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
