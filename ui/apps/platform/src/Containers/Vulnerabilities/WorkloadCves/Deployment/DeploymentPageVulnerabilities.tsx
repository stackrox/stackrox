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
    Tab,
    Tabs,
    TabsComponent,
    TabTitleText,
    Text,
    Title,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import useURLPagination from 'hooks/useURLPagination';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { DynamicTableLabel } from '../components/DynamicIcon';
import WorkloadTableToolbar from '../components/WorkloadTableToolbar';
import TableErrorComponent from '../components/TableErrorComponent';
import { cveStatusTabValues } from '../types';
import BySeveritySummaryCard from '../SummaryCards/BySeveritySummaryCard';
import CvesByStatusSummaryCard, {
    resourceCountByCveSeverityAndStatusFragment,
    ResourceCountByCveSeverityAndStatus,
} from '../SummaryCards/CvesByStatusSummaryCard';
import { parseQuerySearchFilter, getHiddenSeverities, getHiddenStatuses } from '../searchUtils';
import { imageMetadataContextFragment } from '../Tables/table.utils';
import DeploymentVulnerabilitiesTable, {
    deploymentWithVulnerabilitiesFragment,
    DeploymentWithVulnerabilities,
} from '../Tables/DeploymentVulnerabilitiesTable';

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

const vulnerabilityQuery = gql`
    ${imageMetadataContextFragment}
    ${deploymentWithVulnerabilitiesFragment}
    query getCvesForDeployment($id: ID!, $query: String!, $pagination: Pagination!) {
        deployment(id: $id) {
            imageVulnerabilityCount(query: $query)
            ...DeploymentWithVulnerabilities
        }
    }
`;

const defaultSortFields = ['CVE'];

export type DeploymentPageVulnerabilitiesProps = {
    deploymentId: string;
};

function DeploymentPageVulnerabilities({ deploymentId }: DeploymentPageVulnerabilitiesProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('cveStatus', cveStatusTabValues);

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

    const summaryRequest = useQuery<
        {
            deployment: {
                id: string;
                imageCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
            };
        },
        { id: string; query: string }
    >(summaryQuery, {
        variables: {
            id: deploymentId,
            query: getRequestQueryStringForSearchFilter(querySearchFilter),
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
            deployment: DeploymentWithVulnerabilities & {
                imageVulnerabilityCount: number;
            };
        },
        {
            id: string;
            query: string;
            pagination: PaginationParam;
        }
    >(vulnerabilityQuery, {
        variables: {
            id: deploymentId,
            query: getRequestQueryStringForSearchFilter(querySearchFilter),
            pagination,
        },
    });

    const vulnerabilityData = vulnerabilityRequest.data ?? vulnerabilityRequest.previousData;
    const totalVulnerabilityCount = vulnerabilityData?.deployment?.imageVulnerabilityCount ?? 0;

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
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(e, key) => setActiveTabKey(key)}
                    component={TabsComponent.nav}
                    mountOnEnter
                    unmountOnExit
                    isBox
                >
                    <Tab
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        eventKey="Observed"
                        title={<TabTitleText>Observed CVEs</TabTitleText>}
                    >
                        <div className="pf-u-px-sm pf-u-background-color-100">
                            <WorkloadTableToolbar />
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
                                {summaryData && (
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
                                                {pluralize(
                                                    totalVulnerabilityCount,
                                                    'result',
                                                    'results'
                                                )}{' '}
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
                                                if (
                                                    totalVulnerabilityCount <
                                                    (page - 1) * newPerPage
                                                ) {
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
                                {vulnerabilityData && (
                                    <DeploymentVulnerabilitiesTable
                                        deployment={vulnerabilityData.deployment}
                                        getSortParams={getSortParams}
                                        isFiltered={isFiltered}
                                    />
                                )}
                            </div>
                        </div>
                    </Tab>
                    <Tab
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        eventKey="Deferred"
                        title={<TabTitleText>Deferrals</TabTitleText>}
                        isDisabled
                    />
                    <Tab
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        eventKey="False Positive"
                        title={<TabTitleText>False positives</TabTitleText>}
                        isDisabled
                    />
                </Tabs>
            </PageSection>
        </>
    );
}

export default DeploymentPageVulnerabilities;
