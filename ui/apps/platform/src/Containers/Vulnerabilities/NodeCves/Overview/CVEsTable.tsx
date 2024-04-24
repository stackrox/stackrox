import React from 'react';
import { Link } from 'react-router-dom';
import { Text } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import useSet from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import { getTableUIState } from 'utils/getTableUIState';

import TooltipTh from 'Components/TooltipTh';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

import { getNodeEntityPagePath } from '../../utils/searchUtils';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import { getScoreVersionsForTopCVSS, sortCveDistroList } from '../../utils/sortUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { QuerySearchFilter } from '../../types';
import useNodeCves from './useNodeCves';
import useTotalNodeCount from './useTotalNodeCount';

export type CVEsTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
};

function CVEsTable({ querySearchFilter, isFiltered, pagination }: CVEsTableProps) {
    const { page, perPage } = pagination;

    const { data, previousData, loading, error } = useNodeCves(querySearchFilter, page, perPage);
    const totalNodeCount = useTotalNodeCount();

    const tableData = data ?? previousData;
    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.nodeCVEs,
        error,
        searchFilter: querySearchFilter,
    });

    const expandedRowSet = useSet<string>();
    const colSpan = 6;

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
                    <Th aria-label="Expand row" />
                    <Th>CVE</Th>
                    <TooltipTh tooltip="The number of nodes affected by this CVE, grouped by the severity of the CVE on each node">
                        Nodes by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th>Top CVSS</Th>
                    <TooltipTh tooltip="Ratio of the number of nodes affected by this CVE to the total number of nodes">
                        Affected nodes
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{
                    message: 'No CVEs have been detected for nodes across your secured clusters',
                }}
                renderer={({ data }) =>
                    data.map((nodeCve, rowIndex) => {
                        const {
                            cve,
                            nodeCountBySeverity: { critical, important, moderate, low },
                            distroTuples,
                            topCVSS,
                            affectedNodeCount,
                            firstDiscoveredInSystem,
                        } = nodeCve;
                        const isExpanded = expandedRowSet.has(cve);

                        const prioritizedDistros = sortCveDistroList(distroTuples);
                        const summary =
                            prioritizedDistros.length > 0 ? prioritizedDistros[0].summary : '';
                        const scoreVersions = getScoreVersionsForTopCVSS(topCVSS, distroTuples);

                        return (
                            <Tbody key={cve} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(cve),
                                        }}
                                    />
                                    <Td dataLabel="CVE" modifier="nowrap">
                                        <Link to={getNodeEntityPagePath('CVE', cve)}>{cve}</Link>
                                    </Td>
                                    <Td dataLabel="Nodes by severity">
                                        <SeverityCountLabels
                                            criticalCount={critical.total}
                                            importantCount={important.total}
                                            moderateCount={moderate.total}
                                            lowCount={low.total}
                                            // TODO - Add filtered severities once filter toolbar is in place
                                        />
                                    </Td>
                                    <Td dataLabel="Top CVSS">
                                        <CvssFormatted
                                            cvss={topCVSS}
                                            scoreVersion={
                                                scoreVersions.length > 0
                                                    ? scoreVersions.join('/')
                                                    : undefined
                                            }
                                        />
                                    </Td>
                                    <Td dataLabel="Affected nodes">
                                        {affectedNodeCount} / {totalNodeCount} affected nodes
                                    </Td>
                                    <Td dataLabel="First discovered">
                                        <DateDistance date={firstDiscoveredInSystem} />
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={colSpan - 1}>
                                        <ExpandableRowContent>
                                            {summary ? (
                                                <Text>{summary}</Text>
                                            ) : (
                                                <PartialCVEDataAlert />
                                            )}
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default CVEsTable;
