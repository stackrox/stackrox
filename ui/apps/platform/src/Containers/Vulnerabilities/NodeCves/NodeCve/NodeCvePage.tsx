import React from 'react';
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
    Title,
    pluralize,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getTableUIState } from 'utils/getTableUIState';
import { getHasSearchApplied } from 'utils/searchUtils';
import { DynamicTableLabel } from 'Components/DynamicIcon';

import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import {
    getHiddenSeverities,
    getOverviewPagePath,
    getRegexScopedQueryString,
    parseWorkloadQuerySearchFilter,
} from '../../utils/searchUtils';
import CvePageHeader from '../../components/CvePageHeader';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import AffectedNodesTable from './AffectedNodesTable';
import AffectedNodesSummaryCard from './AffectedNodesSummaryCard';
import useAffectedNodes from './useAffectedNodes';
import useNodeCveMetadata from './useNodeCveMetadata';

const nodeCveOverviewCvePath = getOverviewPagePath('Node', { entityTab: 'CVE' });

function NodeCvePage() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);

    // We need to scope all queries to the *exact* CVE name so that we don't accidentally get
    // data that matches a prefix of the CVE name in the nested fields
    const { cveId } = useParams() as { cveId: string };
    const exactCveIdSearchRegex = `^${cveId}$`;
    const query = getRegexScopedQueryString({
        ...querySearchFilter,
        CVE: [exactCveIdSearchRegex],
    });

    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);

    const { metadataRequest, nodeCount, cveData } = useNodeCveMetadata(cveId, query);

    const { affectedNodesRequest, nodeData } = useAffectedNodes(query, page, perPage);

    const nodeCveName = cveData?.cve;

    const tableState = getTableUIState({
        isLoading: affectedNodesRequest.loading,
        error: affectedNodesRequest.error,
        data: nodeData,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageTitle title={`Node CVEs - NodeCVE ${nodeCveName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={nodeCveOverviewCvePath}>Node CVEs</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeCveName ?? (
                            <Skeleton screenreaderText="Loading CVE name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <CvePageHeader data={cveData} />
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-v5-u-flex-grow-1">
                <SummaryCardLayout
                    error={metadataRequest.error}
                    isLoading={metadataRequest.loading}
                >
                    <SummaryCard
                        data={metadataRequest.data}
                        loadingText="Loading affected nodes summary"
                        renderer={({ data }) => (
                            <AffectedNodesSummaryCard
                                affectedNodeCount={nodeCount}
                                totalNodeCount={data.totalNodeCount}
                                operatingSystemCount={data.nodeCVE.distroTuples.length}
                            />
                        )}
                    />
                    <SummaryCard
                        data={metadataRequest.data}
                        loadingText="Loading affected nodes by CVE severity summary"
                        renderer={({ data }) => (
                            <BySeveritySummaryCard
                                title="Nodes by severity"
                                severityCounts={data.nodeCVE.nodeCountBySeverity}
                                hiddenSeverities={hiddenSeverities}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <div className="pf-v5-u-background-color-100 pf-v5-u-flex-grow-1 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2">
                                    {pluralize(nodeCount, 'node')} affected
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                itemCount={nodeCount}
                                perPage={perPage}
                                page={page}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    if (nodeCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <AffectedNodesTable tableState={tableState} />
                </div>
            </PageSection>
        </>
    );
}

export default NodeCvePage;
