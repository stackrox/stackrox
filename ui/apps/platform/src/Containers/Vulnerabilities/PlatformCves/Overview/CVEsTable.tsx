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

import useURLPagination from 'hooks/useURLPagination';
import useMap from 'hooks/useMap';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ApiSortOption } from 'types/search';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { getTableUIState } from 'utils/getTableUIState';

import TooltipTh from 'Components/TooltipTh';
import CvssFormatted from 'Components/CvssFormatted';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

import ExpandRowTh from 'Components/ExpandRowTh';
import { CVE_SORT_FIELD, CVE_TYPE_SORT_FIELD, CVSS_SORT_FIELD } from '../../utils/sortFields';
import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import { getPlatformEntityPagePath } from '../../utils/searchUtils';
import { QuerySearchFilter } from '../../types';
import usePlatformCves from './usePlatformCves';
import { displayCveType } from '../utils/stringUtils';

const totalClusterCountQuery = gql`
    query getTotalClusterCount {
        clusterCount
    }
`;

export const sortFields = [
    CVE_SORT_FIELD,
    CVE_TYPE_SORT_FIELD,
    CVSS_SORT_FIELD,
    // TODO - Needs a BE field implementation
    // AffectedClusters: '',
];

export const defaultSortOption = { field: CVSS_SORT_FIELD, direction: 'desc' } as const;

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
    canSelectRows,
    createRowActions,
    sortOption,
    getSortParams,
    onClearFilters,
}: CVEsTableProps) {
    const { page, perPage } = pagination;

    const { data, previousData, error, loading } = usePlatformCves({
        querySearchFilter,
        page,
        perPage,
        sortOption,
    });
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
            aria-live="polite"
            aria-busy={loading ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
                    <Th sort={getSortParams(CVE_SORT_FIELD)}>CVE</Th>
                    <Th>CVE status</Th>
                    <Th sort={getSortParams(CVE_TYPE_SORT_FIELD)}>CVE type</Th>
                    <Th sort={getSortParams(CVSS_SORT_FIELD)}>CVSS</Th>
                    <TooltipTh
                        tooltip="Ratio of the number of clusters affected by this CVE to the total number of secured clusters"
                        sort={
                            // TODO - Needs a BE field implementation
                            // getSortParams(sortFields.AffectedClusters)
                            undefined
                        }
                    >
                        Affected clusters
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
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
                    message: 'No CVEs have been detected for your secured clusters',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((platformCve, rowIndex) => {
                        const {
                            id,
                            cve,
                            isFixable,
                            cveType,
                            cvss,
                            clusterVulnerability: { summary, scoreVersion },
                            clusterCountByType,
                        } = platformCve;
                        const isExpanded = expandedRowSet.has(cve);

                        const { generic, kubernetes, openshift, openshift4 } = clusterCountByType;
                        const affectedClusterCount = generic + kubernetes + openshift + openshift4;

                        return (
                            <Tbody key={id} isExpanded={isExpanded}>
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
                                        <Link to={getPlatformEntityPagePath('CVE', id)}>{cve}</Link>
                                    </Td>
                                    <Td dataLabel="CVE status">
                                        <VulnerabilityFixableIconText isFixable={isFixable} />
                                    </Td>
                                    <Td dataLabel="CVE type">{displayCveType(cveType)}</Td>
                                    <Td dataLabel="CVSS">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    <Td dataLabel="Affected clusters">
                                        {affectedClusterCount} / {totalClusterCount} affected
                                        clusters
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
