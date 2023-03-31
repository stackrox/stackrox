import React from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    Flex,
    PageSection,
    Pagination,
    Skeleton,
    Spinner,
    Split,
    SplitItem,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { Pagination as PaginationParam } from 'services/types';

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
import AffectedImagesTable, {
    ImageForCve,
    imagesForCveFragment,
} from './Tables/AffectedImagesTable';
import EntityTypeToggleGroup from './EntityTypeToggleGroup';
import AffectedDeploymentsTable from './Tables/AffectedDeploymentsTable';
import { DynamicTableLabel } from './DynamicIcon';

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

export const imageCveAffectedImagesQuery = gql`
    ${imagesForCveFragment}
    # by default, query must include the CVE id
    query getImagesForCVE(
        $query: String
        $imageListPagination: Pagination
        $imageComponentPagination: Pagination
    ) {
        imageCount(query: $query)
        images(query: $query, pagination: $imageListPagination) {
            ...ImagesForCVE
        }
    }
`;

const defaultSortFields = ['Image', 'Severity', 'Fixable', 'Operating System'];
const imageCveEntities = ['Image', 'Deployment'] as const;

function ImageCvePage() {
    const { cveId } = useParams();
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const { page, perPage, setPage, setPerPage } = useURLPagination(25);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Severity',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const [entityTab] = useURLStringUnion('entityTab', imageCveEntities);

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

    const imageDataRequest = useQuery<
        { imageCount: number; images: ImageForCve[] },
        {
            query: string;
            imageListPagination: PaginationParam;
            // TODO If required, fix this
            imageComponentPagination?: PaginationParam;
        }
    >(imageCveAffectedImagesQuery, {
        variables: {
            query: getRequestQueryStringForSearchFilter({
                ...querySearchFilter,
                CVE: cveId,
            }),
            imageListPagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
            // TODO Benchmark whether or not server side pagination is really needed at this
            // level, and if so, fix the implementation here
            imageComponentPagination: undefined,
        },
        skip: entityTab !== 'Image',
    });

    // We generalize the imageData and deploymentData requests here so that we can use most of
    // the same logic for both tables and components in the return value below
    const imageData = imageDataRequest.data ?? imageDataRequest.previousData;
    let tableDataAvailable = false;
    let tableRowCount = 0;
    let tableError: Error | undefined;
    let tableLoading = false;

    if (entityTab === 'Image') {
        tableDataAvailable = !!imageData;
        tableRowCount = imageDataRequest.data?.imageCount ?? 0;
        tableError = imageDataRequest.error;
        tableLoading = imageDataRequest.loading;
    } else if (entityTab === 'Deployment') {
        tableDataAvailable = false;
        tableRowCount = 0;
        tableError = undefined;
        tableLoading = false;
    }

    const cveName = metadataRequest.data?.imageCVE?.cve;

    const isFiltered = getHasSearchApplied(querySearchFilter);
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
                </div>
                <Divider />
                <div className="pf-u-background-color-100 pf-u-flex-grow-1">
                    <Split className="pf-u-px-lg pf-u-py-md pf-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <EntityTypeToggleGroup
                                    imageCount={imageData?.imageCount ?? 0}
                                    entityTabs={imageCveEntities}
                                />
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                isCompact
                                itemCount={tableRowCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    if (tableRowCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    {tableError ? (
                        <Bullseye>
                            <EmptyStateTemplate
                                headingLevel="h2"
                                title={getAxiosErrorMessage(tableError)}
                                icon={ExclamationCircleIcon}
                                iconClassName="pf-u-danger-color-100"
                            >
                                Adjust your filters and try again
                            </EmptyStateTemplate>
                        </Bullseye>
                    ) : (
                        <>
                            {tableLoading && !tableDataAvailable && (
                                <Bullseye>
                                    <Spinner isSVG />
                                </Bullseye>
                            )}
                            {tableDataAvailable && (
                                <>
                                    <Divider />
                                    <div className="pf-u-px-lg">
                                        {entityTab === 'Image' && (
                                            <AffectedImagesTable
                                                images={imageData?.images ?? []}
                                                getSortParams={getSortParams}
                                                isFiltered={isFiltered}
                                            />
                                        )}
                                        {entityTab === 'Deployment' && <AffectedDeploymentsTable />}
                                    </div>
                                </>
                            )}
                        </>
                    )}
                </div>
            </PageSection>
        </>
    );
}

export default ImageCvePage;
