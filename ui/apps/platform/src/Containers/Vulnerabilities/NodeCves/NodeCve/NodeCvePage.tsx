import React, { useEffect, useState } from 'react';
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
    getOverviewPagePath,
    getRegexScopedQueryString,
    parseWorkloadQuerySearchFilter,
} from '../../utils/searchUtils';
import CvePageHeader, { CveMetadata } from '../../components/CvePageHeader';
import { DEFAULT_PAGE_SIZE } from '../../constants';
import AffectedNodesTable, { AffectedNode, affectedNodeFragment } from './AffectedNodesTable';

const workloadCveOverviewCvePath = getOverviewPagePath('Node', {
    entityTab: 'CVE',
});

const affectedNodesQuery = gql`
    ${affectedNodeFragment}
    query getAffectedNodes($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            ...AffectedNode
        }
    }
`;

function NodeCvePage() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);

    // We needs to scope all queries to the *exact* CVE name so that we don't accidentally get
    // data that matches a prefix of the CVE name in the nested fields
    const { cveId } = useParams() as { cveId: string };
    const exactCveIdSearchRegex = `^${cveId}$`;
    const query = getRegexScopedQueryString({
        ...querySearchFilter,
        CVE: [exactCveIdSearchRegex],
    });

    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_PAGE_SIZE);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const [nodeCveMetadata, setNodeCveMetadata] = useState<CveMetadata>();
    const nodeCveName = nodeCveMetadata?.cve;

    // TODO - Simulate a loading state, will replace metadata with results from a query
    useEffect(() => {
        setTimeout(() => {
            setNodeCveMetadata({
                cve: cveId,
                firstDiscoveredInSystem: '2021-01-01T00:00:00Z',
                distroTuples: [
                    {
                        summary: 'This is a sample description used during development',
                        link: `https://access.redhat.com/security/cve/${cveId}`,
                        operatingSystem: 'rhel',
                    },
                ],
            });
        }, 1500);
    }, [cveId]);

    const affectedNodesRequest = useQuery<
        {
            nodes: AffectedNode[];
        },
        {
            query: string;
            pagination: { limit: number; offset: number };
        }
    >(affectedNodesQuery, {
        variables: {
            query,
            pagination: {
                limit: perPage,
                offset: (page - 1) * perPage,
            },
        },
    });

    const nodeData = affectedNodesRequest.data?.nodes ?? affectedNodesRequest.previousData?.nodes;
    const nodeCount = 50; // TODO

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
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>CVEs</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeCveName ?? (
                            <Skeleton screenreaderText="Loading CVE name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <CvePageHeader data={nodeCveMetadata} />
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-v5-u-flex-grow-1">
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
