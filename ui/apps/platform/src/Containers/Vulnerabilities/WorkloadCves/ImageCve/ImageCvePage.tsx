import React from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Pagination,
    Skeleton,
    Spinner,
    Split,
    SplitItem,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage';
import PageTitle from 'Components/PageTitle';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getHasSearchApplied } from 'utils/searchUtils';
import { Pagination as PaginationParam } from 'services/types';

import { VulnerabilitySeverity } from 'types/cve.proto';
import {
    CLUSTER_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    SearchOption,
} from 'Containers/Vulnerabilities/components/SearchOptionsDropdown';
import {
    getHiddenSeverities,
    getOverviewCvesPath,
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
} from '../searchUtils';
import WorkloadTableToolbar from '../components/WorkloadTableToolbar';
import ImageCvePageHeader, {
    ImageCveMetadata,
    imageCveMetadataFragment,
} from './ImageCvePageHeader';
import AffectedImagesTable, {
    ImageForCve,
    imagesForCveFragment,
} from '../Tables/AffectedImagesTable';
import EntityTypeToggleGroup from '../components/EntityTypeToggleGroup';
import { DynamicTableLabel } from '../components/DynamicIcon';
import TableErrorComponent from '../components/TableErrorComponent';
import AffectedDeploymentsTable, {
    DeploymentForCve,
    deploymentsForCveFragment,
} from '../Tables/AffectedDeploymentsTable';
import AffectedImages from '../SummaryCards/AffectedImages';
import BySeveritySummaryCard, {
    ResourceCountsByCveSeverity,
} from '../SummaryCards/BySeveritySummaryCard';
import { resourceCountByCveSeverityAndStatusFragment } from '../SummaryCards/CvesByStatusSummaryCard';
import { VulnerabilitySeverityLabel } from '../types';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

const workloadCveOverviewCvePath = getOverviewCvesPath({
    vulnerabilityState: 'OBSERVED',
    entityTab: 'CVE',
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
    ${resourceCountByCveSeverityAndStatusFragment}
    query getImageCveSummaryData($cve: String!, $query: String!) {
        totalImageCount: imageCount
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVE(cve: $cve, subfieldScopeQuery: $query) {
            cve
            affectedImageCount
            affectedImageCountBySeverity {
                ...ResourceCountsByCVESeverityAndStatus
            }
        }
    }
`;

export const imageCveAffectedImagesQuery = gql`
    ${imagesForCveFragment}
    # by default, query must include the CVE id
    query getImagesForCVE(
        $query: String
        $pagination: Pagination
        $statusesForExceptionCount: [String!]
    ) {
        images(query: $query, pagination: $pagination) {
            ...ImagesForCVE
        }
    }
`;

export const imageCveAffectedDeploymentsQuery = gql`
    ${deploymentsForCveFragment}
    # by default, query must include the CVE id
    query getDeploymentsForCVE(
        $query: String
        $pagination: Pagination
        $lowImageCountQuery: String
        $moderateImageCountQuery: String
        $importantImageCountQuery: String
        $criticalImageCountQuery: String
        $statusesForExceptionCount: [String!]
    ) {
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

const defaultSeveritySummary = {
    affectedImageCountBySeverity: {
        critical: { total: 0 },
        important: { total: 0 },
        moderate: { total: 0 },
        low: { total: 0 },
    },
    affectedImageCount: 0,
    topCVSS: 0,
};

const searchOptions: SearchOption[] = [
    IMAGE_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
];

function ImageCvePage() {
    const currentVulnerabilityState = useVulnerabilityState();

    const urlParams = useParams();
    const cveId: string = urlParams.cveId ?? '';
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const query = getVulnStateScopedQueryString(
        {
            ...querySearchFilter,
            CVE: [cveId],
        },
        currentVulnerabilityState
    );
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);

    const [entityTab] = useURLStringUnion('entityTab', imageCveEntities);

    const { sortOption, setSortOption, getSortParams } = useURLSort({
        sortFields: getSortFields(entityTab),
        defaultSortOption: getDefaultSortOption(entityTab),
        onSort: () => setPage(1),
    });

    const metadataRequest = useQuery<{ imageCVE: ImageCveMetadata | null }, { cve: string }>(
        imageCveMetadataQuery,
        { variables: { cve: cveId } }
    );

    const summaryRequest = useQuery<
        {
            totalImageCount: number;
            imageCount: number;
            deploymentCount: number;
            imageCVE: {
                affectedImageCountBySeverity: ResourceCountsByCveSeverity;
                affectedImageCount: number;
            };
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

    function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
        const filters = { ...querySearchFilter, CVE: [cveId] };
        if (severity) {
            filters.Severity = [severity];
        }
        return getVulnStateScopedQueryString(filters, currentVulnerabilityState);
    }

    const deploymentDataRequest = useQuery<
        { deploymentCount: number; deployments: DeploymentForCve[] },
        {
            query: string;
            lowImageCountQuery: string;
            moderateImageCountQuery: string;
            importantImageCountQuery: string;
            criticalImageCountQuery: string;
            pagination: PaginationParam;
        }
    >(imageCveAffectedDeploymentsQuery, {
        variables: {
            query: getDeploymentSearchQuery(),
            lowImageCountQuery: getDeploymentSearchQuery('LOW_VULNERABILITY_SEVERITY'),
            moderateImageCountQuery: getDeploymentSearchQuery('MODERATE_VULNERABILITY_SEVERITY'),
            importantImageCountQuery: getDeploymentSearchQuery('IMPORTANT_VULNERABILITY_SEVERITY'),
            criticalImageCountQuery: getDeploymentSearchQuery('CRITICAL_VULNERABILITY_SEVERITY'),
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

    // If the `imageCVE` field is null, then the CVE ID passed via URL does not exist
    if (metadataRequest.data && metadataRequest.data.imageCVE === null) {
        return (
            <NotFoundMessage
                title="404: We couldn't find that page"
                message={`A CVE with ID ${cveId} could not be found.`}
            />
        );
    }

    const cveName = metadataRequest.data?.imageCVE?.cve;

    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const severitySummary = summaryRequest.data?.imageCVE ?? defaultSeveritySummary;

    return (
        <>
            <PageTitle
                title={`Workload CVEs - ImageCVE ${metadataRequest.data?.imageCVE?.cve ?? ''}`}
            />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>CVEs</BreadcrumbItemLink>
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
                    <ImageCvePageHeader data={metadataRequest.data?.imageCVE ?? undefined} />
                )}
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1">
                <VulnerabilityStateTabs titleOverrides={{ observed: 'Workloads' }} isBox />
                <div className="pf-u-background-color-100">
                    <div className="pf-u-px-sm">
                        <WorkloadTableToolbar
                            searchOptions={searchOptions}
                            autocompleteSearchContext={{
                                'CVE ID': cveId,
                            }}
                            onFilterChange={() => setPage(1)}
                        />
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
                        {!summaryRequest.error && summaryRequest.data && (
                            <Grid hasGutter>
                                <GridItem sm={12} md={6} xl2={4}>
                                    <AffectedImages
                                        className="pf-u-h-100"
                                        affectedImageCount={severitySummary.affectedImageCount}
                                        totalImagesCount={summaryRequest.data.totalImageCount}
                                    />
                                </GridItem>
                                <GridItem sm={12} md={6} xl2={4}>
                                    <BySeveritySummaryCard
                                        className="pf-u-h-100"
                                        title="Images by severity"
                                        severityCounts={
                                            severitySummary.affectedImageCountBySeverity
                                        }
                                        hiddenSeverities={hiddenSeverities}
                                    />
                                </GridItem>
                            </Grid>
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
                                    setSortOption={setSortOption}
                                    setPage={setPage}
                                />
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
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
                                    <div className="pf-u-px-lg workload-cves-table-container">
                                        {entityTab === 'Image' && (
                                            <AffectedImagesTable
                                                images={imageData?.images ?? []}
                                                getSortParams={getSortParams}
                                                isFiltered={isFiltered}
                                                cve={cveId}
                                                vulnerabilityState={currentVulnerabilityState}
                                            />
                                        )}
                                        {entityTab === 'Deployment' && (
                                            <AffectedDeploymentsTable
                                                deployments={deploymentData?.deployments ?? []}
                                                getSortParams={getSortParams}
                                                isFiltered={isFiltered}
                                                filteredSeverities={
                                                    searchFilter.Severity as VulnerabilitySeverityLabel[]
                                                }
                                                cve={cveId}
                                                vulnerabilityState={currentVulnerabilityState}
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
