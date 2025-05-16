import React from 'react';
import {
    Divider,
    Flex,
    PageSection,
    Pagination,
    pluralize,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied, getPaginationParams } from 'utils/searchUtils';
import NotFoundMessage from 'Components/NotFoundMessage';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import { getTableUIState } from 'utils/getTableUIState';
import AdvancedFiltersToolbar from 'Containers/Vulnerabilities/components/AdvancedFiltersToolbar';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import {
    flattenImageComponentSearchFilterConfig,
    flattenImageCVESearchFilterConfig,
    imageSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import { filterManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import CvesByStatusSummaryCard, {
    resourceCountByCveSeverityAndStatusFragment,
    ResourceCountByCveSeverityAndStatus,
} from '../SummaryCards/CvesByStatusSummaryCard';
import {
    parseQuerySearchFilter,
    getHiddenSeverities,
    getHiddenStatuses,
    getVulnStateScopedQueryString,
    getStatusesForExceptionCount,
} from '../../utils/searchUtils';
import {
    DeploymentWithVulnerabilities,
    formatVulnerabilityData,
    imageMetadataContextFragment,
} from '../Tables/table.utils';
import DeploymentVulnerabilitiesTable, {
    defaultColumns,
    deploymentWithVulnerabilitiesFragment,
    tableId,
} from '../Tables/DeploymentVulnerabilitiesTable';
import VulnerabilityStateTabs, {
    vulnStateTabContentId,
} from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

const summaryQuery = gql`
    ${resourceCountByCveSeverityAndStatusFragment}
    query getDeploymentSummaryData($id: ID!, $query: String!) {
        deployment(id: $id) {
            id
            imageCVECountBySeverity(query: $query) {
                ...ResourceCountsByCVESeverityAndStatus
            }
        }
    }
`;

export const deploymentVulnerabilitiesQuery = gql`
    ${imageMetadataContextFragment}
    ${deploymentWithVulnerabilitiesFragment}
    query getCvesForDeployment(
        $id: ID!
        $query: String!
        $pagination: Pagination!
        $statusesForExceptionCount: [String!]
    ) {
        deployment(id: $id) {
            imageVulnerabilityCount(query: $query)
            ...DeploymentWithVulnerabilities
        }
    }
`;

const defaultSortFields = ['CVE', 'Severity'];

export type DeploymentPageVulnerabilitiesProps = {
    deploymentId: string;
    pagination: UseURLPaginationResult;
};

function DeploymentPageVulnerabilities({
    deploymentId,
    pagination,
}: DeploymentPageVulnerabilitiesProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { baseSearchFilter } = useWorkloadCveViewContext();

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    const { page, setPage, perPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Severity',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);

    const query = getVulnStateScopedQueryString(
        { ...baseSearchFilter, ...querySearchFilter },
        currentVulnerabilityState
    );

    const summaryRequest = useQuery<
        {
            deployment: {
                id: string;
                imageCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
            } | null;
        },
        { id: string; query: string; statusesForExceptionCount: string[] }
    >(summaryQuery, {
        fetchPolicy: 'no-cache',
        nextFetchPolicy: 'no-cache',
        variables: {
            id: deploymentId,
            query,
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
    });

    const summaryData = summaryRequest.data ?? summaryRequest.previousData;

    const vulnerabilityRequest = useQuery<
        {
            deployment:
                | (DeploymentWithVulnerabilities & {
                      imageVulnerabilityCount: number;
                  })
                | null;
        },
        {
            id: string;
            query: string;
            pagination: PaginationParam;
            statusesForExceptionCount: string[];
        }
    >(deploymentVulnerabilitiesQuery, {
        fetchPolicy: 'no-cache',
        nextFetchPolicy: 'no-cache',
        variables: {
            id: deploymentId,
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
    });

    // Omit for 4.7 release until CVE/advisory separation is available in 4.8 release.
    // const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = false;
    const filteredColumns = filterManagedColumns(
        defaultColumns,
        (key) => key !== 'epssProbability' || isEpssProbabilityColumnEnabled
    );
    const managedColumnState = useManagedColumns(tableId, filteredColumns);

    const isFlattenCveDataEnabled = isFeatureFlagEnabled('ROX_FLATTEN_CVE_DATA');
    const imageCVESearchFilterConfig = flattenImageCVESearchFilterConfig(isFlattenCveDataEnabled);
    const searchFilterConfigWithFeatureFlagDependency = [
        imageSearchFilterConfig,
        // Omit EPSSProbability for 4.7 release until CVE/advisory separation is available in 4.8 release.
        // imageCVESearchFilterConfig,
        {
            ...imageCVESearchFilterConfig,
            attributes: imageCVESearchFilterConfig.attributes.filter(
                ({ searchTerm }) => searchTerm !== 'EPSS Probability'
            ),
        },
        flattenImageComponentSearchFilterConfig(isFlattenCveDataEnabled),
    ];

    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfigWithFeatureFlagDependency
    );

    const vulnerabilityData = vulnerabilityRequest.data ?? vulnerabilityRequest.previousData;
    const totalVulnerabilityCount = vulnerabilityData?.deployment?.imageVulnerabilityCount ?? 0;

    const deploymentNotFound =
        (vulnerabilityData && !vulnerabilityData.deployment) ||
        (summaryData && !summaryData.deployment);

    if (deploymentNotFound) {
        return (
            <NotFoundMessage
                title="404: We couldn't find that page"
                message={`A deployment with ID ${deploymentId} could not be found.`}
            />
        );
    }

    const tableState = getTableUIState({
        isLoading: vulnerabilityRequest.loading,
        data: vulnerabilityRequest?.data?.deployment
            ? formatVulnerabilityData(vulnerabilityRequest.data.deployment)
            : undefined,
        error: vulnerabilityRequest.error,
        searchFilter,
    });

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>
                    Review and triage vulnerability data scanned for images within this deployment
                </Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                id={vulnStateTabContentId}
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                <VulnerabilityStateTabs
                    isBox
                    onChange={() => {
                        setSearchFilter({});
                        setPage(1);
                    }}
                />
                <div className="pf-v5-u-px-sm pf-v5-u-background-color-100">
                    <AdvancedFiltersToolbar
                        className="pf-v5-u-pt-lg pf-v5-u-pb-0"
                        searchFilterConfig={searchFilterConfig}
                        searchFilter={searchFilter}
                        onFilterChange={(newFilter, searchPayload) => {
                            setSearchFilter(newFilter);
                            setPage(1);
                            trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
                        }}
                        additionalContextFilter={{
                            'Deployment ID': deploymentId,
                            ...baseSearchFilter,
                        }}
                    />
                </div>
                <SummaryCardLayout error={summaryRequest.error} isLoading={summaryRequest.loading}>
                    <SummaryCard
                        data={summaryData?.deployment}
                        loadingText="Loading deployment summary data"
                        renderer={({ data }) => (
                            <BySeveritySummaryCard
                                title="CVEs by severity"
                                severityCounts={data.imageCVECountBySeverity}
                                hiddenSeverities={hiddenSeverities}
                            />
                        )}
                    />
                    <SummaryCard
                        data={summaryData?.deployment}
                        loadingText="Loading deployment summary data"
                        renderer={({ data }) => (
                            <CvesByStatusSummaryCard
                                cveStatusCounts={data.imageCVECountBySeverity}
                                hiddenStatuses={hiddenStatuses}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider />
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100">
                    <div className="pf-v5-u-p-lg">
                        <Split hasGutter className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                            <SplitItem isFilled>
                                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                    <Title headingLevel="h2">
                                        {pluralize(totalVulnerabilityCount, 'result', 'results')}{' '}
                                        found
                                    </Title>
                                    {isFiltered && <DynamicTableLabel />}
                                </Flex>
                            </SplitItem>
                            <SplitItem>
                                <ColumnManagementButton managedColumnState={managedColumnState} />
                            </SplitItem>
                            <SplitItem>
                                <Pagination
                                    itemCount={totalVulnerabilityCount}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => {
                                        setPerPage(newPerPage);
                                    }}
                                />
                            </SplitItem>
                        </Split>
                        <div className="workload-cves-table-container">
                            <DeploymentVulnerabilitiesTable
                                tableState={tableState}
                                getSortParams={getSortParams}
                                isFiltered={isFiltered}
                                vulnerabilityState={currentVulnerabilityState}
                                onClearFilters={() => {
                                    setSearchFilter({});
                                    setPage(1);
                                }}
                                tableConfig={managedColumnState.columns}
                            />
                        </div>
                    </div>
                </div>
            </PageSection>
        </>
    );
}

export default DeploymentPageVulnerabilities;
