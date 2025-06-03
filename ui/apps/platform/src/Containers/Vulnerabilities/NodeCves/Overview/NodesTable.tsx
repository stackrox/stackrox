import React from 'react';
import { Link } from 'react-router-dom';
import { Truncate } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import DateDistance from 'Components/DateDistance';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import { getTableUIState } from 'utils/getTableUIState';
import useURLPagination from 'hooks/useURLPagination';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ApiSortOption } from 'types/search';

import { vulnerabilitySeverityLabels } from 'messages/common';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

import {
    CLUSTER_SORT_FIELD,
    NODE_SCAN_TIME_SORT_FIELD,
    NODE_SORT_FIELD,
    OPERATING_SYSTEM_SORT_FIELD,
} from '../../utils/sortFields';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import { QuerySearchFilter, isVulnerabilitySeverityLabel } from '../../types';
import useNodes from './useNodes';

export const sortFields = [
    NODE_SORT_FIELD,
    CLUSTER_SORT_FIELD,
    OPERATING_SYSTEM_SORT_FIELD,
    NODE_SCAN_TIME_SORT_FIELD,
];

export const defaultSortOption = { field: NODE_SORT_FIELD, direction: 'asc' } as const;

export type NodesTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function NodesTable({
    querySearchFilter,
    isFiltered,
    pagination,
    sortOption,
    getSortParams,
    onClearFilters,
}: NodesTableProps) {
    const { page, perPage } = pagination;

    const { data, previousData, loading, error } = useNodes({
        querySearchFilter,
        page,
        perPage,
        sortOption,
    });
    const tableData = data ?? previousData;

    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.nodes,
        error,
        searchFilter: querySearchFilter,
    });

    const filteredSeverities = querySearchFilter.SEVERITY?.map(
        (s) => vulnerabilitySeverityLabels[s]
    ).filter(isVulnerabilitySeverityLabel);

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={loading ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams(NODE_SORT_FIELD)}>Node</Th>
                    <Th>
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams(CLUSTER_SORT_FIELD)}>Cluster</Th>
                    <Th sort={getSortParams(OPERATING_SYSTEM_SORT_FIELD)}>Operating system</Th>
                    <Th sort={getSortParams(NODE_SCAN_TIME_SORT_FIELD)}>Scan time</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={5}
                emptyProps={{ message: 'No CVEs have been reported for your scanned nodes' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((node) => {
                            const { id, name, nodeCVECountBySeverity, cluster, osImage, scanTime } =
                                node;
                            const { critical, important, moderate, low, unknown } =
                                nodeCVECountBySeverity;
                            return (
                                <Tr key={id}>
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
                                            unknownCount={unknown.total}
                                            filteredSeverities={filteredSeverities}
                                            entity={'node'}
                                        />
                                    </Td>
                                    <Td dataLabel="Cluster" modifier="nowrap">
                                        <Truncate position="middle" content={cluster.name} />
                                    </Td>
                                    <Td dataLabel="Operating system" modifier="nowrap">
                                        <Truncate position="middle" content={osImage} />
                                    </Td>
                                    <Td dataLabel="Scan time">
                                        <DateDistance date={scanTime} />
                                    </Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default NodesTable;
