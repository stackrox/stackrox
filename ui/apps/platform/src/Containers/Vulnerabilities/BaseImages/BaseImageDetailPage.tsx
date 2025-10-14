import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    PageSection,
    Skeleton,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { useParams } from 'react-router-dom-v5-compat';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { vulnerabilitiesBasePath } from 'routePaths';
import BaseImageHeader from './components/BaseImageHeader';
import BaseImageCVEsTab from './tabs/BaseImageCVEsTab';
import { useBaseImages } from './hooks/useBaseImages';

/**
 * Base Image detail page - shows comprehensive information about a tracked base image
 */
function BaseImageDetailPage() {
    const { id } = useParams<{ id: string }>();
    const { getBaseImageById } = useBaseImages();

    const baseImage = getBaseImageById(id || '');
    const baseImagesListPath = `${vulnerabilitiesBasePath}/base-images`;

    if (!id) {
        return (
            <>
                <PageTitle title="Base Image Details" />
                <PageSection variant="light">
                    <Bullseye>
                        <EmptyStateTemplate
                            title="No base image ID provided"
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-v5-u-danger-color-100"
                        />
                    </Bullseye>
                </PageSection>
            </>
        );
    }

    if (!baseImage) {
        return (
            <>
                <PageTitle title="Base Image Details" />
                <PageSection variant="light">
                    <Bullseye>
                        <EmptyStateTemplate
                            title="Base image not found"
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-v5-u-danger-color-100"
                        >
                            The base image with ID &quot;{id}&quot; could not be found. It may have
                            been removed or the ID is invalid.
                        </EmptyStateTemplate>
                    </Bullseye>
                </PageSection>
            </>
        );
    }

    return (
        <>
            <PageTitle title={`Base Images - ${baseImage.name}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={baseImagesListPath}>Base Images</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {baseImage ? (
                            baseImage.name
                        ) : (
                            <Skeleton screenreaderText="Loading base image name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />

            <BaseImageHeader baseImage={baseImage} />

            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                padding={{ default: 'noPadding' }}
            >
                <BaseImageCVEsTab baseImageId={id} />
            </PageSection>
        </>
    );
}

export default BaseImageDetailPage;
