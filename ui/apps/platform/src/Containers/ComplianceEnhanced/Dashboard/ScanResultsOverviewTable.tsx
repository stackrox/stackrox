import React, { useEffect, useState } from 'react';
import {
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
    ActionsColumn,
    IAction,
} from '@patternfly/react-table';
import { format } from 'date-fns';

import usePagination from 'hooks/patternfly/usePagination';
import useTableSort from 'hooks/patternfly/useTableSort';

import { ComplianceScanResultsOverview } from './types';
import ScanResultsToolbar, { SearchFilter } from './ScanResultsToolbar';

import scanResultsOverviewMockData from './MockData/complianceScanResultsOverview.json';

const defaultSearchFilter = {
    scanName: '',
    clusterName: '',
    profileName: '',
};

const sortFields = ['Scan Name', 'Failing Controls', 'Last Scanned'];
const defaultSortOption = { field: 'Last Scanned', direction: 'asc' } as const;

function ScanResultsOverviewTable() {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const [searchFilter, setSearchFilter] = useState<SearchFilter>(defaultSearchFilter);
    const [scanResultsOverviewData] = useState<ComplianceScanResultsOverview[]>(
        scanResultsOverviewMockData.scanOverviews
    );

    useEffect(() => {
        // TODO: fetch GetComplianceScanResultsOverview once implemented and update scanResultsOverviewData
    }, [searchFilter, page, perPage, sortOption]);

    const defaultActions = (): IAction[] => [
        {
            title: 'Edit schedule',
            // TODO: link to edit schedule page
        },
    ];

    const displayOnlyItemOrItemCount = (items: string[], multipleItemLabel: string): string => {
        if (items.length > 1) {
            return `${items.length} ${multipleItemLabel}`;
        }
        return items[0];
    };

    return (
        <>
            <ScanResultsToolbar
                numberOfScanResults={scanResultsOverviewData.length}
                searchFilter={searchFilter}
                setSearchFilter={setSearchFilter}
                page={page}
                perPage={perPage}
                onSetPage={onSetPage}
                onPerPageSelect={onPerPageSelect}
            />
            <TableComposable borders={false}>
                <Thead noWrap>
                    <Tr>
                        <Th sort={getSortParams('Scan Name')}>Scan</Th>
                        <Th>Clusters</Th>
                        <Th>Profiles</Th>
                        <Th sort={getSortParams('Failing Controls')}>Failing Controls</Th>
                        <Th sort={getSortParams('Last Scanned')}>Last Scanned</Th>
                        <Td />
                    </Tr>
                </Thead>
                <Tbody>
                    {scanResultsOverviewData.map(({ scanStats, clusterId, profileName }) => {
                        return (
                            <Tr key={scanStats.scanName}>
                                <Td>{scanStats.scanName}</Td> {/* TODO: link to scan details  */}
                                <Td>{displayOnlyItemOrItemCount(clusterId, 'clusters')}</Td>
                                <Td>{displayOnlyItemOrItemCount(profileName, 'profiles')}</Td>
                                <Td>{`${scanStats.numberOfFailingChecks}/${scanStats.numberOfChecks}`}</Td>
                                <Td>{format(scanStats.lastScan, 'DD MMM YYYY, h:mm:ss A')}</Td>
                                <Td isActionCell>
                                    <ActionsColumn items={defaultActions()} />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
        </>
    );
}

export default ScanResultsOverviewTable;
