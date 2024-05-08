import React from 'react';
import { Link } from 'react-router-dom';
import { Text } from '@patternfly/react-core';
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    Table,
    Thead,
    Tr,
    Th,
    Tbody,
    Td,
} from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';
import sum from 'lodash/sum';

import useURLPagination from 'hooks/useURLPagination';
import useMap from 'hooks/useMap';
import useSet from 'hooks/useSet';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { getTableUIState } from 'utils/getTableUIState';

import TooltipTh from 'Components/TooltipTh';
import CvssFormatted from 'Components/CvssFormatted';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import { sortCveDistroList } from '../../utils/sortUtils';
import { getPlatformEntityPagePath } from '../../utils/searchUtils';
import { QuerySearchFilter } from '../../types';
import usePlatformCves from './usePlatformCves';

const totalClusterCountQuery = gql`
    query getTotalClusterCount {
        clusterCount
    }
`;

export type CVEsTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
    selectedCves: ReturnType<typeof useMap<string, { cve: string }>>;
    createRowActions: (cve: { cve: string }) => IAction[];
    canSelectRows?: boolean;
};

function CVEsTable({
    querySearchFilter,
    isFiltered,
    pagination,
    selectedCves,
    canSelectRows,
    createRowActions,
}: CVEsTableProps) {
    const { page, perPage } = pagination;

    const { data, previousData, error, loading } = usePlatformCves(
        querySearchFilter,
        page,
        perPage
    );
    const totalClusterCountRequest = useQuery(totalClusterCountQuery);
    const totalClusterCount = totalClusterCountRequest.data?.clusterCount ?? 0;

    const tableData = data ?? previousData;

    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.platformCVEs,
        error,
        searchFilter: querySearchFilter,
    });

    const expandedRowSet = useSet<string>();
    const colSpan = canSelectRows ? 8 : 6;

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
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
                    <Th>CVE</Th>
                    <Th>CVE status</Th>
                    <Th>CVE type</Th>
                    <Th>CVSS</Th>
                    <TooltipTh tooltip="Ratio of the number of clusters affected by this CVE to the total number of secured clusters">
                        Affected clusters
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    {canSelectRows && <Th aria-label="CVE actions" />}
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{
                    message: 'No CVEs have been detected for your secured clusters',
                }}
                renderer={({ data }) =>
                    data.map((platformCve, rowIndex) => {
                        const {
                            cve,
                            isFixable,
                            cveType,
                            cvss,
                            scoreVersion,
                            distroTuples,
                            clusterCountByType,
                        } = platformCve;
                        const isExpanded = expandedRowSet.has(cve);

                        const prioritizedDistros = sortCveDistroList(distroTuples);
                        const summary =
                            prioritizedDistros.length > 0 ? prioritizedDistros[0].summary : '';
                        const affectedClusterCount = sum(Object.values(clusterCountByType));

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
                                        <Link to={getPlatformEntityPagePath('CVE', cve)}>
                                            {cve}
                                        </Link>
                                    </Td>
                                    <Td dataLabel="CVE status">
                                        <VulnerabilityFixableIconText isFixable={isFixable} />
                                    </Td>
                                    <Td dataLabel="CVE type">{cveType}</Td>
                                    <Td dataLabel="CVSS">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    <Td dataLabel="Affected clusters">
                                        {affectedClusterCount} / {totalClusterCount} affected
                                        clusters
                                    </Td>
                                    {canSelectRows && (
                                        <Td className="pf-v5-u-px-0">
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
