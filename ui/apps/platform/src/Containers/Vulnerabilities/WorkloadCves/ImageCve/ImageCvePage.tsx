import React, { useEffect } from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    PageSection,
    Pagination,
    Skeleton,
    Split,
    SplitItem,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage';
import PageTitle from 'Components/PageTitle';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getHasSearchApplied, getPaginationParams } from 'utils/searchUtils';
import { Pagination as PaginationParam } from 'services/types';

import { VulnerabilitySeverity } from 'types/cve.proto';
import useAnalytics, {
    WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
    WORKLOAD_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import { getTableUIState } from 'utils/getTableUIState';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageComponentSearchFilterConfig,
    imageSearchFilterConfig,
    namespaceSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import { filterManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import { WorkloadEntityTab, VulnerabilitySeverityLabel } from '../../types';
import {
    getHiddenSeverities,
    getOverviewPagePath,
    getStatusesForExceptionCount,
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import CvePageHeader, { CveMetadata } from '../../components/CvePageHeader';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

import AffectedImagesTable, {
    ImageForCve,
    convertToFlatImagesForCveFragment, // imagesForCveFragment
    tableId as affectedImagesTableId,
    defaultColumns as affectedImagesDefaultColumns,
} from '../Tables/AffectedImagesTable';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import AffectedDeploymentsTable, {
    DeploymentForCve,
    convertToFlatDeploymentsForCveFragment, // deploymentsForCveFragment
    tableId as affectedDeploymentsTableId,
    defaultColumns as affectedDeploymentsDefaultColumns,
} from '../Tables/AffectedDeploymentsTable';
import AffectedImages from '../SummaryCards/AffectedImages';
import BySeveritySummaryCard, {
    ResourceCountsByCveSeverity,
} from '../../components/BySeveritySummaryCard';
import { resourceCountByCveSeverityAndStatusFragment } from '../SummaryCards/CvesByStatusSummaryCard';
import VulnerabilityStateTabs, {
    vulnStateTabContentId,
} from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export const imageCveMetadataQuery = gql`
    query getImageCveMetadata($cve: String!) {
        imageCVE(cve: $cve) {
            cve
            firstDiscoveredInSystem
            publishedOn
            distroTuples {
                summary
                link
                operatingSystem
                cveBaseInfo {
                    epss {
                        epssProbability
                    }
                }
            }
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

// After release, replace temporary function
// with imageCveAffectedImagesQuery
// that has unconditional imagesForCveFragment.
export function convertToFlatImageCveAffectedImagesQuery(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
) {
    return gql`
        ${convertToFlatImagesForCveFragment(isFlattenCveDataEnabled)}
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
}

// After release, replace temporary function
// with imageCveAffectedDeploymentsQuery
// that has unconditional deploymentsForCveFragment.
export function convertToFlatImageCveAffectedDeploymentsQuery(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
) {
    return gql`
        ${convertToFlatDeploymentsForCveFragment(isFlattenCveDataEnabled)}
        # by default, query must include the CVE id
        query getDeploymentsForCVE(
            $query: String
            $pagination: Pagination
            $unknownImageCountQuery: String
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
}

const imageSortFields = ['Image', 'Severity', 'Operating System'];
const imageDefaultSort = { field: 'Severity', direction: 'desc' } as const;

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
        unknown: { total: 0 },
    },
    affectedImageCount: 0,
    topCVSS: 0,
};

const searchFilterConfig = [
    imageSearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
];

function ImageCvePage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { getAbsoluteUrl, pageTitle, baseSearchFilter } = useWorkloadCveViewContext();
    const currentVulnerabilityState = useVulnerabilityState();

    const urlParams = useParams();
    const cveId = urlParams.cveId ?? '';
    const exactCveIdSearchRegex = `^${cveId}$`;
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const query = getVulnStateScopedQueryString(
        {
            CVE: [exactCveIdSearchRegex],
            ...baseSearchFilter,
            ...querySearchFilter,
        },
        currentVulnerabilityState
    );
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const [entityTab] = useURLStringUnion('entityTab', imageCveEntities);

    const { sortOption, setSortOption, getSortParams } = useURLSort({
        sortFields: getSortFields(entityTab),
        defaultSortOption: getDefaultSortOption(entityTab),
        onSort: () => setPage(1),
    });

    const metadataRequest = useQuery<{ imageCVE: CveMetadata | null }, { cve: string }>(
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

    const isFlattenCveDataEnabled = isFeatureFlagEnabled('ROX_FLATTEN_CVE_DATA');
    const imageCveAffectedImagesQuery =
        convertToFlatImageCveAffectedImagesQuery(isFlattenCveDataEnabled);
    const imageDataRequest = useQuery<
        { images: ImageForCve[] },
        {
            query: string;
            pagination: PaginationParam;
            statusesForExceptionCount: string[];
        }
    >(imageCveAffectedImagesQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
        skip: entityTab !== 'Image',
    });

    function getDeploymentSearchQuery(severity?: VulnerabilitySeverity) {
        const filters = { CVE: [exactCveIdSearchRegex], ...baseSearchFilter, ...querySearchFilter };
        if (severity) {
            filters.SEVERITY = [severity];
        }
        return getVulnStateScopedQueryString(filters, currentVulnerabilityState);
    }

    const imageCveAffectedDeploymentsQuery =
        convertToFlatImageCveAffectedDeploymentsQuery(isFlattenCveDataEnabled);
    const deploymentDataRequest = useQuery<
        { deploymentCount: number; deployments: DeploymentForCve[] },
        {
            query: string;
            unknownImageCountQuery: string;
            lowImageCountQuery: string;
            moderateImageCountQuery: string;
            importantImageCountQuery: string;
            criticalImageCountQuery: string;
            pagination: PaginationParam;
            statusesForExceptionCount: string[];
        }
    >(imageCveAffectedDeploymentsQuery, {
        variables: {
            query: getDeploymentSearchQuery(),
            unknownImageCountQuery: getDeploymentSearchQuery('UNKNOWN_VULNERABILITY_SEVERITY'),
            lowImageCountQuery: getDeploymentSearchQuery('LOW_VULNERABILITY_SEVERITY'),
            moderateImageCountQuery: getDeploymentSearchQuery('MODERATE_VULNERABILITY_SEVERITY'),
            importantImageCountQuery: getDeploymentSearchQuery('IMPORTANT_VULNERABILITY_SEVERITY'),
            criticalImageCountQuery: getDeploymentSearchQuery('CRITICAL_VULNERABILITY_SEVERITY'),
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
        skip: entityTab !== 'Deployment',
    });

    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const affectedImagesFilteredColumns = filterManagedColumns(
        affectedImagesDefaultColumns,
        (key) => key !== 'nvdCvss' || isNvdCvssColumnEnabled
    );
    const imageTableColumnState = useManagedColumns(
        affectedImagesTableId,
        affectedImagesFilteredColumns
    );

    const deploymentTableColumnState = useManagedColumns(
        affectedDeploymentsTableId,
        affectedDeploymentsDefaultColumns
    );

    const imageCount = summaryRequest.data?.imageCount ?? 0;
    const deploymentCount = summaryRequest.data?.deploymentCount ?? 0;

    let tableRowCount = 0;

    if (entityTab === 'Image') {
        tableRowCount = imageCount;
    } else if (entityTab === 'Deployment') {
        tableRowCount = deploymentCount;
    }

    function onEntityTypeChange(entityTab: WorkloadEntityTab) {
        setPage(1);
        if (entityTab !== 'CVE') {
            setSortOption(getDefaultSortOption(entityTab));
        }
        analyticsTrack({
            event: WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
            properties: {
                type: entityTab,
                page: 'CVE Detail',
            },
        });
    }

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    // Track the initial entity tab view
    useEffect(() => {
        onEntityTypeChange(entityTab);
    }, []);

    // If the `imageCVE` field is null, then the CVE ID passed via URL does not exist
    if (metadataRequest.data && metadataRequest.data.imageCVE === null) {
        return (
            <NotFoundMessage
                title="404: We couldn't find that page"
                message={`A CVE with ID ${cveId} could not be found.`}
            />
        );
    }

    const workloadCveOverviewCvePath = getAbsoluteUrl(
        getOverviewPagePath('Workload', {
            vulnerabilityState: 'OBSERVED',
            entityTab: 'CVE',
        })
    );

    const cveName = metadataRequest.data?.imageCVE?.cve;

    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const severitySummary = summaryRequest.data?.imageCVE ?? defaultSeveritySummary;

    const imageTableState = getTableUIState({
        isLoading: imageDataRequest.loading,
        data: imageDataRequest.data?.images,
        error: imageDataRequest.error,
        searchFilter,
    });
    const deploymentTableState = getTableUIState({
        isLoading: deploymentDataRequest.loading,
        data: deploymentDataRequest.data?.deployments,
        error: deploymentDataRequest.error,
        searchFilter,
    });

    return (
        <>
            <PageTitle
                title={`${pageTitle} - ImageCVE ${metadataRequest.data?.imageCVE?.cve ?? ''}`}
            />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>CVEs</BreadcrumbItemLink>
                    {!metadataRequest.error && (
                        <BreadcrumbItem isActive>
                            {cveName ?? (
                                <Skeleton screenreaderText="Loading CVE name" width="200px" />
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
                    <CvePageHeader data={metadataRequest.data?.imageCVE ?? undefined} />
                )}
            </PageSection>
            <Divider component="div" />
            <PageSection
                id={vulnStateTabContentId}
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
            >
                <VulnerabilityStateTabs
                    titleOverrides={{ observed: 'Workloads' }}
                    isBox
                    onChange={() => {
                        setSearchFilter({});
                        setPage(1);
                    }}
                />
                <div className="pf-v5-u-background-color-100">
                    <div className="pf-v5-u-px-sm">
                        <AdvancedFiltersToolbar
                            className="pf-v5-u-py-md"
                            searchFilterConfig={searchFilterConfig}
                            searchFilter={searchFilter}
                            onFilterChange={(newFilter, searchPayload) => {
                                setSearchFilter(newFilter);
                                setPage(1);
                                trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
                            }}
                            additionalContextFilter={{
                                // Only allow exact match for CVE ID using quotes, the autocomplete API does not
                                // support regex for exact matching
                                CVE: `"${cveId}"`,
                                ...baseSearchFilter,
                            }}
                        />
                    </div>
                    <SummaryCardLayout
                        error={summaryRequest.error}
                        isLoading={summaryRequest.loading}
                    >
                        <SummaryCard
                            data={summaryRequest.data}
                            loadingText="Loading image CVE summary data"
                            renderer={({ data }) => (
                                <AffectedImages
                                    className="pf-v5-u-h-100"
                                    affectedImageCount={severitySummary.affectedImageCount}
                                    totalImagesCount={data.totalImageCount}
                                />
                            )}
                        />
                        <SummaryCard
                            data={severitySummary}
                            loadingText="Loading image CVE summary data"
                            renderer={({ data }) => (
                                <BySeveritySummaryCard
                                    title="Images by severity"
                                    severityCounts={data.affectedImageCountBySeverity}
                                    hiddenSeverities={hiddenSeverities}
                                />
                            )}
                        />
                    </SummaryCardLayout>
                </div>
                <Divider />
                <div className="pf-v5-u-background-color-100 pf-v5-u-flex-grow-1">
                    <Split
                        hasGutter
                        className="pf-v5-u-px-lg pf-v5-u-py-md pf-v5-u-align-items-baseline"
                    >
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <EntityTypeToggleGroup
                                    entityTabs={imageCveEntities}
                                    entityCounts={{
                                        Image: imageCount,
                                        Deployment: deploymentCount,
                                    }}
                                    onChange={onEntityTypeChange}
                                />
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            {entityTab === 'Image' && (
                                <ColumnManagementButton
                                    managedColumnState={imageTableColumnState}
                                />
                            )}
                            {entityTab === 'Deployment' && (
                                <ColumnManagementButton
                                    managedColumnState={deploymentTableColumnState}
                                />
                            )}
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                itemCount={tableRowCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <Divider />
                    <div className="pf-v5-u-px-lg workload-cves-table-container">
                        {entityTab === 'Image' && (
                            <AffectedImagesTable
                                tableState={imageTableState}
                                getSortParams={getSortParams}
                                isFiltered={isFiltered}
                                cve={cveId}
                                vulnerabilityState={currentVulnerabilityState}
                                onClearFilters={onClearFilters}
                                tableConfig={imageTableColumnState.columns}
                            />
                        )}
                        {entityTab === 'Deployment' && (
                            <AffectedDeploymentsTable
                                tableState={deploymentTableState}
                                getSortParams={getSortParams}
                                isFiltered={isFiltered}
                                filteredSeverities={
                                    searchFilter.SEVERITY as VulnerabilitySeverityLabel[]
                                }
                                cve={cveId}
                                vulnerabilityState={currentVulnerabilityState}
                                onClearFilters={onClearFilters}
                                tableConfig={deploymentTableColumnState.columns}
                            />
                        )}
                    </div>
                </div>
            </PageSection>
        </>
    );
}

export default ImageCvePage;
