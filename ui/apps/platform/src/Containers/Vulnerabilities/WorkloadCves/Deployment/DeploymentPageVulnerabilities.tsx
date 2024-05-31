import React from 'react';
import {
    Bullseye,
    Divider,
    Flex,
    PageSection,
    Pagination,
    pluralize,
    Spinner,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied } from 'utils/searchUtils';
import NotFoundMessage from 'Components/NotFoundMessage';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import {
    SearchOption,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
} from '../../searchOptions';
import WorkloadCveFilterToolbar from '../components/WorkloadCveFilterToolbar';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import CvesByStatusSummaryCard, {
    resourceCountByCveSeverityAndStatusFragment,
    ResourceCountByCveSeverityAndStatus,
} from '../SummaryCards/CvesByStatusSummaryCard';
import {
    parseWorkloadQuerySearchFilter,
    getHiddenSeverities,
    getHiddenStatuses,
    getVulnStateScopedQueryString,
    getStatusesForExceptionCount,
} from '../../utils/searchUtils';
import { imageMetadataContextFragment } from '../Tables/table.utils';
import DeploymentVulnerabilitiesTable, {
    deploymentWithVulnerabilitiesFragment,
    DeploymentWithVulnerabilities,
} from '../Tables/DeploymentVulnerabilitiesTable';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

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

const defaultSortFields = ['CVE'];

const searchOptions: SearchOption[] = [
    IMAGE_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
];

export type DeploymentPageVulnerabilitiesProps = {
    deploymentId: string;
    pagination: UseURLPaginationResult;
};

function DeploymentPageVulnerabilities({
    deploymentId,
    pagination,
}: DeploymentPageVulnerabilitiesProps) {
    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);

    const { page, setPage, perPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'CVE',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);

    const query = getVulnStateScopedQueryString(querySearchFilter, currentVulnerabilityState);

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
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
    });

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

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>
                    Review and triage vulnerability data scanned for images within this deployment
                </Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                <VulnerabilityStateTabs isBox onChange={() => setPage(1)} />
                <div className="pf-v5-u-px-sm pf-v5-u-background-color-100">
                    <WorkloadCveFilterToolbar
                        autocompleteSearchContext={{
                            'Deployment ID': deploymentId,
                        }}
                        searchOptions={searchOptions}
                        onFilterChange={() => setPage(1)}
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
                                isBusy={summaryRequest.loading}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider />
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100">
                    <div className="pf-v5-u-p-lg">
                        <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
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
                                <Pagination
                                    isCompact
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
                        {vulnerabilityRequest.error && (
                            <TableErrorComponent
                                error={vulnerabilityRequest.error}
                                message="Adjust your filters and try again"
                            />
                        )}
                        {vulnerabilityRequest.loading && !vulnerabilityData && (
                            <Bullseye>
                                <Spinner />
                            </Bullseye>
                        )}
                        {vulnerabilityData && vulnerabilityData.deployment && (
                            <div className="workload-cves-table-container">
                                <DeploymentVulnerabilitiesTable
                                    deployment={vulnerabilityData.deployment}
                                    getSortParams={getSortParams}
                                    isFiltered={isFiltered}
                                    vulnerabilityState={currentVulnerabilityState}
                                />
                            </div>
                        )}
                    </div>
                </div>
            </PageSection>
        </>
    );
}

export default DeploymentPageVulnerabilities;
