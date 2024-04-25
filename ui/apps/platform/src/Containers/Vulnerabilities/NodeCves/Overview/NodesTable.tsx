import React from 'react';
import { Link } from 'react-router-dom';
import { Truncate } from '@patternfly/react-core';
import { Table, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';

import DateDistance from 'Components/DateDistance';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import {
    TbodyLoading,
    TbodyError,
    TbodyEmpty,
    TbodyFilteredEmpty,
} from 'Components/TableStateTemplates';
import { getTableUIState } from 'utils/getTableUIState';
import useURLPagination from 'hooks/useURLPagination';

import { vulnerabilitySeverityLabels } from 'messages/common';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { getNodeEntityPagePath, getRegexScopedQueryString } from '../../utils/searchUtils';
import { QuerySearchFilter, isVulnerabilitySeverityLabel } from '../../types';

const nodeListQuery = gql`
    query getNodes($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            id
            name
            nodeCVECountBySeverity {
                critical {
                    total
                }
                important {
                    total
                }
                moderate {
                    total
                }
                low {
                    total
                }
            }
            cluster {
                name
            }
            operatingSystem
            scanTime
        }
    }
`;

// TODO - Verify these types once the BE is implemented
type Node = {
    id: string;
    name: string;
    nodeCVECountBySeverity: {
        critical: {
            total: number;
        };
        important: {
            total: number;
        };
        moderate: {
            total: number;
        };
        low: {
            total: number;
        };
    };
    cluster: {
        name: string;
    };
    operatingSystem: string;
    scanTime: string;
};

export type NodesTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
};

function NodesTable({ querySearchFilter, isFiltered, pagination }: NodesTableProps) {
    const { page, perPage } = pagination;
    const { data, previousData, loading, error } = useQuery<{ nodes: Node[] }>(nodeListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
            },
        },
    });

    const tableData = data ?? previousData;
    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.nodes,
        error,
        searchFilter: querySearchFilter,
    });
    const colSpan = 5;

    const filteredSeverities = querySearchFilter.SEVERITY?.map(
        (s) => vulnerabilitySeverityLabels[s]
    ).filter(isVulnerabilitySeverityLabel);

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            role="region"
            aria-live="polite"
            aria-busy={loading ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th>Node</Th>
                    <Th>
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>Cluster</Th>
                    <Th>Operating system</Th>
                    <Th>Scan time</Th>
                </Tr>
            </Thead>
            {tableState.type === 'LOADING' && <TbodyLoading colSpan={colSpan} />}
            {tableState.type === 'ERROR' && (
                <TbodyError colSpan={colSpan} error={tableState.error} />
            )}
            {tableState.type === 'EMPTY' && (
                <TbodyEmpty
                    colSpan={colSpan}
                    message="No CVEs have been reported for your scanned nodes"
                />
            )}
            {tableState.type === 'FILTERED_EMPTY' && <TbodyFilteredEmpty colSpan={colSpan} />}
            {tableState.type === 'COMPLETE' &&
                tableState.data.map((node) => {
                    const { id, name, nodeCVECountBySeverity, cluster, operatingSystem, scanTime } =
                        node;
                    const { critical, important, moderate, low } = nodeCVECountBySeverity;
                    return (
                        <Tr key={node.id}>
                            <Td dataLabel="Node" modifier="nowrap">
                                <Link to={getNodeEntityPagePath('Node', id)}>
                                    <Truncate position="middle" content={name} />
                                </Link>
                            </Td>
                            <Td dataLabel="CVEs by severity">
                                <SeverityCountLabels
                                    criticalCount={critical.total}
                                    importantCount={important.total}
                                    moderateCount={moderate.total}
                                    lowCount={low.total}
                                    filteredSeverities={filteredSeverities}
                                />
                            </Td>
                            <Td dataLabel="Cluster" modifier="nowrap">
                                <Truncate position="middle" content={cluster.name} />
                            </Td>
                            <Td dataLabel="Operating system" modifier="nowrap">
                                {operatingSystem}
                            </Td>
                            <Td dataLabel="Scan time">
                                <DateDistance date={scanTime} />
                            </Td>
                        </Tr>
                    );
                })}
        </Table>
    );
}

export default NodesTable;
