import React from 'react';
import {
    Flex,
    PageSection,
    Pagination,
    Skeleton,
    Split,
    SplitItem,
    Text,
    Title,
    pluralize,
} from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import { getHasSearchApplied, getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { getTableUIState } from 'utils/getTableUIState';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import { SummaryCardLayout, SummaryCard } from '../../components/SummaryCardLayout';
import { getHiddenStatuses, parseWorkloadQuerySearchFilter } from '../../utils/searchUtils';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

import useClusterVulnerabilities from './useClusterVulnerabilities';
import useClusterSummaryData from './useClusterSummaryData';
import CVEsTable from './CVEsTable';
import PlatformCvesByStatusSummaryCard from './PlatformCvesByStatusSummaryCard';
import PlatformCvesByTypeSummaryCard from './PlatformCvesByTypeSummaryCard';

export type ClusterPageVulnerabilitiesProps = {
    clusterId: string;
};

function ClusterPageVulnerabilities({ clusterId }: ClusterPageVulnerabilitiesProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const query = getUrlQueryStringForSearchFilter(querySearchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const { data, loading, error } = useClusterVulnerabilities(clusterId, query, page, perPage);
    const summaryRequest = useClusterSummaryData(clusterId, query);

    const hiddenStatuses = getHiddenStatuses(querySearchFilter);
    const clusterVulnerabilityCount = data?.cluster.clusterVulnerabilityCount ?? 0;

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.cluster.clusterVulnerabilities,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this cluster</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <SummaryCardLayout isLoading={summaryRequest.loading} error={summaryRequest.error}>
                    <SummaryCard
                        loadingText={'Loading platform CVEs by status summary'}
                        data={summaryRequest.data}
                        renderer={({ data }) => (
                            <PlatformCvesByStatusSummaryCard
                                data={data.cluster.platformCVECountByFixability}
                                hiddenStatuses={hiddenStatuses}
                            />
                        )}
                    />
                    <SummaryCard
                        loadingText={'Loading platform CVEs by type summary'}
                        data={summaryRequest.data}
                        renderer={({ data }) => (
                            <PlatformCvesByTypeSummaryCard
                                data={data.cluster.platformCVECountByType}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2" className="pf-v5-u-w-50">
                                    {data ? (
                                        `${pluralize(clusterVulnerabilityCount, 'result')} found`
                                    ) : (
                                        <Skeleton screenreaderText="Loading cluster vulnerability count" />
                                    )}
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                itemCount={clusterVulnerabilityCount}
                                perPage={perPage}
                                page={page}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    if (clusterVulnerabilityCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <CVEsTable tableState={tableState} />
                </div>
            </PageSection>
        </>
    );
}

export default ClusterPageVulnerabilities;
