import React from 'react';
import { Link } from 'react-router-dom';
import { Truncate } from '@patternfly/react-core';
import { Table, Td, Th, Thead, Tr } from '@patternfly/react-table';

import DateDistance from 'Components/DateDistance';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import { getTableUIState } from 'utils/getTableUIState';
import useURLPagination from 'hooks/useURLPagination';

import { vulnerabilitySeverityLabels } from 'messages/common';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import { QuerySearchFilter, isVulnerabilitySeverityLabel } from '../../types';
import useNodes from './useNodes';

export type NodesTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
};

function NodesTable({ querySearchFilter, isFiltered, pagination }: NodesTableProps) {
    const { page, perPage } = pagination;

    const { data, previousData, loading, error } = useNodes(querySearchFilter, page, perPage);
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
            <TbodyUnified
                tableState={tableState}
                colSpan={5}
                emptyProps={{ message: 'No CVEs have been reported for your scanned nodes' }}
                renderer={({ data }) =>
                    data.map((node) => {
                        const {
                            id,
                            name,
                            nodeCVECountBySeverity,
                            cluster,
                            operatingSystem,
                            scanTime,
                        } = node;
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
                    })
                }
            />
        </Table>
    );
}

export default NodesTable;
