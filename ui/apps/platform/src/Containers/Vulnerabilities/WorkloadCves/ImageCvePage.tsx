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
import PageTitle from 'Components/PageTitle';
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
import EntityTypeToggleGroup from './components/EntityTypeToggleGroup';
import { DynamicTableLabel } from './components/DynamicIcon';
import TableErrorComponent from './components/TableErrorComponent';
import AffectedDeploymentsTable, {
    DeploymentForCve,
    deploymentsForCveFragment,
} from './Tables/AffectedDeploymentsTable';

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
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVE(cve: $cve) {
            cve
            ...ImageCVESeveritySummary
        }
    }
`;

export const imageCveAffectedImagesQuery = gql`
    ${imagesForCveFragment}
    # by default, query must include the CVE id
    query getImagesForCVE($query: String, $pagination: Pagination) {
        images(query: $query, pagination: $pagination) {
            ...ImagesForCVE
        }
    }
`;

export const imageCveAffectedDeploymentsQuery = gql`
    ${deploymentsForCveFragment}
    # by default, query must include the CVE id
    query getDeploymentsForCVE($query: String, $pagination: Pagination) {
        deployments(query: $query, pagination: $pagination) {
            ...DeploymentsForCVE
        }
    }
`;

const imageSortFields = ['Image', 'Operating System'];
const imageDefaultSort = { field: 'Image', direction: 'desc' } as const;

const deploymentSortFields = ['Deployment', 'Cluster', 'Namespace'];
const deploymentDefaultSort = { field: 'Deployment', direction: 'desc' } as const;

const imageCveEntities = ['Image', 'Deployment'] as const;

function getSortFields(entityTab: (typeof imageCveEntities)[number]) {
    return entityTab === 'Image' ? imageSortFields : deploymentSortFields;
}

function getDefaultSortOption(entityTab: (typeof imageCveEntities)[number]) {
    return entityTab === 'Image' ? imageDefaultSort : deploymentDefaultSort;
}

function ImageCvePage() {
    const { cveId } = useParams();
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const query = getRequestQueryStringForSearchFilter({
        ...querySearchFilter,
        CVE: cveId,
    });
    const { page, perPage, setPage, setPerPage } = useURLPagination(25);

    const [entityTab] = useURLStringUnion('entityTab', imageCveEntities);

    const { sortOption, setSortOption, getSortParams } = useURLSort({
        sortFields: getSortFields(entityTab),
        defaultSortOption: getDefaultSortOption(entityTab),
        onSort: () => setPage(1),
    });

    const metadataRequest = useQuery<{ imageCVE: ImageCveMetadata }, { cve: string }>(
        imageCveMetadataQuery,
        { variables: { cve: cveId } }
    );

    const summaryRequest = useQuery<
        ImageCveSummaryCount & {
            imageCount: number;
            deploymentCount: number;
            imageCVE: ImageCveSeveritySummary;
        },
        { cve: string; query: string }
    >(imageCveSummaryQuery, {
        variables: {
            cve: cveId,
            query,
        },
    });

    const imageDataRequest = useQuery<
        { images: ImageForCve[] },
        {
            query: string;
            pagination: PaginationParam;
        }
    >(imageCveAffectedImagesQuery, {
        variables: {
            query,
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
        skip: entityTab !== 'Image',
    });

    const deploymentDataRequest = useQuery<
        { deploymentCount: number; deployments: DeploymentForCve[] },
        {
            query: string;
            pagination: PaginationParam;
        }
    >(imageCveAffectedDeploymentsQuery, {
        variables: {
            query: getRequestQueryStringForSearchFilter({
                ...querySearchFilter,
                CVE: cveId,
            }),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
        skip: entityTab !== 'Deployment',
    });

    // We generalize the imageData and deploymentData requests here so that we can use most of
    // the same logic for both tables and components in the return value below
    const imageData = imageDataRequest.data ?? imageDataRequest.previousData;
    const deploymentData = deploymentDataRequest.data ?? deploymentDataRequest.previousData;
    const imageCount = summaryRequest.data?.imageCount ?? 0;
    const deploymentCount = summaryRequest.data?.deploymentCount ?? 0;

    let tableDataAvailable = false;
    let tableRowCount = 0;
    let tableError: Error | undefined;
    let tableLoading = false;

    if (entityTab === 'Image') {
        tableDataAvailable = !!imageData;
        tableRowCount = imageCount;
        tableError = imageDataRequest.error;
        tableLoading = imageDataRequest.loading;
    } else if (entityTab === 'Deployment') {
        tableDataAvailable = !!deploymentData;
        tableRowCount = deploymentCount;
        tableError = deploymentDataRequest.error;
        tableLoading = deploymentDataRequest.loading;
    }

    const cveName = metadataRequest.data?.imageCVE?.cve;

    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);

    return (
        <>
            <PageTitle
                title={`Workload CVEs - ImageCVE ${metadataRequest.data?.imageCVE.cve ?? ''}`}
            />
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
                    <TableErrorComponent
                        error={metadataRequest.error}
                        message="The system was unable to load metadata for this CVE"
                    />
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
                                    imageCount={imageCount}
                                    deploymentCount={deploymentCount}
                                    entityTabs={imageCveEntities}
                                    onChange={(entity) => {
                                        // Ugly type workaround
                                        if (entity !== 'CVE') {
                                            // Set the sort and pagination back to the default when changing between entity tabs
                                            setSortOption(getDefaultSortOption(entity));
                                            setPage(1);
                                        }
                                    }}
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
                        <TableErrorComponent
                            error={tableError}
                            message="Adjust your filters and try again"
                        />
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
                                        {entityTab === 'Deployment' && (
                                            <AffectedDeploymentsTable
                                                deployments={deploymentData?.deployments ?? []}
                                                getSortParams={getSortParams}
                                                isFiltered={isFiltered}
                                            />
                                        )}
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
