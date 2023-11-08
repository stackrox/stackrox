import React, { useCallback } from 'react';
import {
    ActionsColumn,
    IAction,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { Bullseye, Spinner } from '@patternfly/react-core';
import { format } from 'date-fns';
import { Link, generatePath } from 'react-router-dom';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import { complianceEnhancedStatusScansPath } from 'routePaths';
import { complianceResultsOverview } from 'services/ComplianceEnhancedService';
import { SortOption } from 'types/table';
import { displayOnlyItemOrItemCount } from 'utils/textUtils';

import ScanResultsToolbar from './ScanResultsToolbar';

const sortFields = ['Scan Name', 'Failing Controls', 'Last Scanned'];
const defaultSortOption = { field: 'Scan Name', direction: 'asc' } as SortOption;

function ScanResultsOverviewTable() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const listQuery = useCallback(
        () => complianceResultsOverview(searchFilter, sortOption, page - 1, perPage),
        [searchFilter, sortOption, page, perPage]
    );
    const { data: scanResultsOverviewData, loading: isLoading } = useRestQuery(listQuery);

    const defaultActions = (): IAction[] => [
        {
            title: 'Edit schedule',
            // TODO: link to edit schedule page
        },
    ];

    const renderTableContent = () => {
        if (isLoading) {
            return (
                <Tr>
                    <Td colSpan={8}>
                        <Bullseye>
                            <Spinner isSVG />
                        </Bullseye>
                    </Td>
                </Tr>
            );
        }

        return scanResultsOverviewData?.map(({ scanStats, clusterId, profileName }) => (
            <Tr key={scanStats.id}>
                <Td>
                    <Link
                        to={generatePath(complianceEnhancedStatusScansPath, { id: scanStats.id })}
                        className="your-button-class-name-if-any"
                    >
                        {scanStats.scanName}
                    </Link>
                </Td>
                <Td>{displayOnlyItemOrItemCount(clusterId, 'clusters')}</Td>
                <Td>{displayOnlyItemOrItemCount(profileName, 'profiles')}</Td>
                <Td>{`${scanStats.numberOfFailingChecks}/${scanStats.numberOfChecks}`}</Td>
                <Td>{format(scanStats.lastScan, 'DD MMM YYYY, h:mm:ss A')}</Td>
                <Td isActionCell>
                    <ActionsColumn items={defaultActions()} />
                </Td>
            </Tr>
        ));
    };

    return (
        <>
            <ScanResultsToolbar
                numberOfScanResults={
                    scanResultsOverviewData ? scanResultsOverviewData.length : null
                }
                searchFilter={searchFilter}
                setSearchFilter={(value) => {
                    setPage(1);
                    setSearchFilter(value);
                }}
                page={page}
                perPage={perPage}
                setPage={setPage}
                setPerPage={setPerPage}
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
                <Tbody>{renderTableContent()}</Tbody>
            </TableComposable>
        </>
    );
}

export default ScanResultsOverviewTable;
