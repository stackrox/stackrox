import React from 'react';
import { Link } from 'react-router-dom';
import { Text } from '@patternfly/react-core';
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import useSet from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import { getTableUIState } from 'utils/getTableUIState';

import TooltipTh from 'Components/TooltipTh';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useMap from 'hooks/useMap';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ApiSortOption } from 'types/search';

import ExpandRowTh from 'Components/ExpandRowTh';
import { vulnerabilitySeverityLabels } from 'messages/common';
import {
    CVE_SORT_FIELD,
    NODE_COUNT_SORT_FIELD,
    NODE_TOP_CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import CVESelectionTd from '../../components/CVESelectionTd';
import CVESelectionTh from '../../components/CVESelectionTh';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import {
    aggregateByCVSS,
    aggregateByDistinctCount,
    getScoreVersionsForTopCVSS,
    sortCveDistroList,
} from '../../utils/sortUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { QuerySearchFilter, isVulnerabilitySeverityLabel } from '../../types';
import useNodeCves from './useNodeCves';
import useTotalNodeCount from './useTotalNodeCount';

export const sortFields = [
    CVE_SORT_FIELD,
    NODE_TOP_CVSS_SORT_FIELD,
    NODE_COUNT_SORT_FIELD,
    // TODO - Needs a BE field implementation
    // FIRST_DISCOVERED_SORT_FIELD,
];

export const defaultSortOption = {
    field: NODE_TOP_CVSS_SORT_FIELD,
    direction: 'desc',
    aggregateBy: {
        aggregateFunc: 'max',
        distinct: 'false',
    },
} as const;

export type CVEsTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
    selectedCves: ReturnType<typeof useMap<string, { cve: string }>>;
    createRowActions: (cve: { cve: string }) => IAction[];
    canSelectRows?: boolean;
    sortOption: ApiSortOption;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function CVEsTable({
    querySearchFilter,
    isFiltered,
    pagination,
    selectedCves,
    createRowActions,
    canSelectRows,
    sortOption,
    getSortParams,
    onClearFilters,
}: CVEsTableProps) {
    const { page, perPage } = pagination;

    const { data, previousData, loading, error } = useNodeCves({
        querySearchFilter,
        page,
        perPage,
        sortOption,
    });
    const totalNodeCount = useTotalNodeCount();

    const tableData = data ?? previousData;
    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.nodeCVEs,
        error,
        searchFilter: querySearchFilter,
    });

    const expandedRowSet = useSet<string>();
    const colSpan = canSelectRows ? 8 : 6;

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
                    <ExpandRowTh />
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
                    <Th sort={getSortParams(CVE_SORT_FIELD)}>CVE</Th>
                    <TooltipTh tooltip="The number of nodes affected by this CVE, grouped by the severity of the CVE on each node">
                        Nodes by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th sort={getSortParams(NODE_TOP_CVSS_SORT_FIELD, aggregateByCVSS)}>
                        Top CVSS
                    </Th>
                    <TooltipTh
                        tooltip="Ratio of the number of nodes affected by this CVE to the total number of nodes"
                        sort={getSortParams('Node ID', aggregateByDistinctCount)}
                    >
                        Affected nodes
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th>First discovered</Th>
                    {canSelectRows && (
                        <Th>
                            <span className="pf-v5-screen-reader">Row actions</span>
                        </Th>
                    )}
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{
                    message: 'No CVEs have been detected for nodes across your secured clusters',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((nodeCve, rowIndex) => {
                        const {
                            cve,
                            affectedNodeCountBySeverity: {
                                critical,
                                important,
                                moderate,
                                low,
                                unknown,
                            },
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
                                    {canSelectRows && (
                                        <CVESelectionTd
                                            selectedCves={selectedCves}
                                            rowIndex={rowIndex}
                                            item={{ cve }}
                                        />
                                    )}
                                    <Td dataLabel="CVE" modifier="nowrap">
                                        <Link to={getNodeEntityPagePath('CVE', cve)}>{cve}</Link>
                                    </Td>
                                    <Td dataLabel="Nodes by severity">
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
                                    {canSelectRows && (
                                        <Td isActionCell>
                                            <ActionsColumn items={createRowActions({ cve })} />
                                        </Td>
                                    )}
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
