import React from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    PageSection,
    Skeleton,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useURLSearch from 'hooks/useURLSearch';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { getHiddenSeverities, getOverviewCvesPath, parseQuerySearchFilter } from './searchUtils';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import ImageCvePageHeader, {
    ImageCveMetadata,
    imageCveMetadataFragment,
} from './ImageCvePageHeader';
import ImageCveSummaryCards, {
    ImageCveSeveritySummary,
    imageCveSeveritySummaryFragment,
    ImageCveSummaryCount,
    imageCveSummaryCountFragment,
} from './ImageCveSummaryCards';

const workloadCveOverviewImagePath = getOverviewCvesPath({
    cveStatusTab: 'Observed',
    entityTab: 'Image',
});

export const imageCveMetadataQuery = gql`
    ${imageCveMetadataFragment}
    query getImageCveMetadata($cve: String!) {
        imageCVE(cve: $cve) {
            ...ImageCVEMetadata
        }
    }
`;

export const imageCveSummaryQuery = gql`
    ${imageCveSummaryCountFragment}
    ${imageCveSeveritySummaryFragment}
    query getImageCveSummaryData($cve: String!, $query: String!) {
        ...ImageCVESummaryCounts
        imageCVE(cve: $cve) {
            cve
            ...ImageCVESeveritySummary
        }
    }
`;

function ImageCvePage() {
    const { cveId } = useParams();
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const metadataRequest = useQuery<{ imageCVE: ImageCveMetadata }, { cve: string }>(
        imageCveMetadataQuery,
        { variables: { cve: cveId } }
    );

    const summaryRequest = useQuery<
        ImageCveSummaryCount & { imageCVE: ImageCveSeveritySummary },
        { cve: string; query: string }
    >(imageCveSummaryQuery, {
        variables: { cve: cveId, query: getRequestQueryStringForSearchFilter(querySearchFilter) },
    });

    const cveName = metadataRequest.data?.imageCVE?.cve;

    const hiddenSeverities = getHiddenSeverities(querySearchFilter);

    return (
        <>
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewImagePath}>CVEs</BreadcrumbItemLink>
                    {!metadataRequest.error && (
                        <BreadcrumbItem isActive>
                            {cveName ?? (
                                <Skeleton screenreaderText="Loading image name" width="200px" />
                            )}
                        </BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                {metadataRequest.error ? (
                    <Bullseye>
                        <EmptyStateTemplate
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-u-danger-color-100"
                            title={getAxiosErrorMessage(metadataRequest.error)}
                        >
                            The system was unable to load metadata for this CVE
                        </EmptyStateTemplate>
                    </Bullseye>
                ) : (
                    // Don't check the loading state here, since if the passed `data` is `undefined` we
                    // will implicitly handle the loading state in the component
                    <ImageCvePageHeader data={metadataRequest.data?.imageCVE} />
                )}
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1">
                <div className="pf-u-background-color-100">
                    <div className="pf-u-px-sm">
                        <WorkloadTableToolbar />
                    </div>
                    <div className="pf-u-px-lg pf-u-pb-lg">
                        {summaryRequest.error && (
                            <Alert
                                title="There was an error loading the summary data for this CVE"
                                isInline
                                variant="danger"
                            >
                                {getAxiosErrorMessage(summaryRequest.error)}
                            </Alert>
                        )}
                        {summaryRequest.loading && !summaryRequest.data && (
                            <Skeleton
                                style={{ height: '120px' }}
                                screenreaderText="Loading image cve summary data"
                            />
                        )}
                        {summaryRequest.data && (
                            <ImageCveSummaryCards
                                summaryCounts={summaryRequest.data}
                                severitySummary={summaryRequest.data.imageCVE}
                                hiddenSeverities={hiddenSeverities}
                            />
                        )}
                    </div>
                    <Divider />
                </div>
            </PageSection>
        </>
    );
}

export default ImageCvePage;
