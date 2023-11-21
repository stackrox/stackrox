import React from 'react';
import {
    Alert,
    Bullseye,
    Divider,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Pagination,
    pluralize,
    Skeleton,
    Spinner,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied } from 'utils/searchUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import NotFoundMessage from 'Components/NotFoundMessage';
import {
    SearchOption,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
} from 'Containers/Vulnerabilities/searchOptions';
import { DynamicTableLabel } from '../components/DynamicIcon';
import WorkloadTableToolbar from '../components/WorkloadTableToolbar';
import TableErrorComponent from '../components/TableErrorComponent';
import BySeveritySummaryCard from '../SummaryCards/BySeveritySummaryCard';
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
} from '../searchUtils';
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
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
];

export type DeploymentPageVulnerabilitiesProps = {
    deploymentId: string;
};

function DeploymentPageVulnerabilities({ deploymentId }: DeploymentPageVulnerabilitiesProps) {
    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    const { page, setPage, perPage, setPerPage } = useURLPagination(20);
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
        variables: {
            id: deploymentId,
            query,
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
    });

    const summaryData = summaryRequest.data ?? summaryRequest.previousData;

    const pagination = {
        offset: (page - 1) * perPage,
        limit: perPage,
        sortOption,
    };

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
        variables: {
            id: deploymentId,
            query,
            pagination,
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
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>
                    Review and triage vulnerability data scanned for images within this deployment
                </Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                component="div"
            >
                <VulnerabilityStateTabs isBox onChange={() => setPage(1)} />
                <div className="pf-u-px-sm pf-u-background-color-100">
                    <WorkloadTableToolbar
                        autocompleteSearchContext={{
                            'Deployment ID': deploymentId,
                        }}
                        searchOptions={searchOptions}
                        onFilterChange={() => setPage(1)}
                    />
                </div>
                <div className="pf-u-flex-grow-1 pf-u-background-color-100">
                    <div className="pf-u-px-lg pf-u-pb-lg">
                        {summaryRequest.error && (
                            <Alert
                                title="There was an error loading the summary data for this deployment"
                                isInline
                                variant="danger"
                            >
                                {getAxiosErrorMessage(summaryRequest.error)}
                            </Alert>
                        )}
                        {summaryRequest.loading && !summaryData && (
                            <Skeleton
                                style={{ height: '120px' }}
                                screenreaderText="Loading deployment summary data"
                            />
                        )}
                        {!summaryRequest.error && summaryData && summaryData.deployment && (
                            <Grid hasGutter>
                                <GridItem sm={12} md={6} xl2={4}>
                                    <BySeveritySummaryCard
                                        title="CVEs by severity"
                                        severityCounts={
                                            summaryData.deployment.imageCVECountBySeverity
                                        }
                                        hiddenSeverities={hiddenSeverities}
                                    />
                                </GridItem>
                                <GridItem sm={12} md={6} xl2={4}>
                                    <CvesByStatusSummaryCard
                                        cveStatusCounts={
                                            summaryData.deployment.imageCVECountBySeverity
                                        }
                                        hiddenStatuses={hiddenStatuses}
                                    />
                                </GridItem>
                            </Grid>
                        )}
                    </div>
                    <Divider />
                    <div className="pf-u-p-lg">
                        <Split className="pf-u-pb-lg pf-u-align-items-baseline">
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
                                        if (totalVulnerabilityCount < (page - 1) * newPerPage) {
                                            setPage(1);
                                        }
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
                                <Spinner isSVG />
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
