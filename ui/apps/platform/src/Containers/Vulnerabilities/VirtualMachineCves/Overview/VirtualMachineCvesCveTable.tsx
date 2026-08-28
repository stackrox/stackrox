import { useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import { listVMCVEs } from 'services/VirtualMachineService';
import type { SearchFilter } from 'types/search';
import { getTableUIState } from 'utils/getTableUIState';

import SeverityCountLabels from '../../components/SeverityCountLabels';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import { getVirtualMachineEntityPagePath } from '../../utils/searchUtils';
import { formatEpssProbabilityAsPercent } from '../../WorkloadCves/Tables/table.utils';
import VirtualMachineCvesFilterToolbar from './VirtualMachineCvesFilterToolbar';
import VirtualMachineCvesToggleGroup from './VirtualMachineCvesToggleGroup';

type VirtualMachineCvesCveTableProps = {
    searchFilter: SearchFilter;
    pagination: UseURLPaginationResult;
    isFiltered: boolean;
    onClearFilters: () => void;
};

function VirtualMachineCvesCveTable({
    searchFilter,
    pagination,
    isFiltered,
    onClearFilters,
}: VirtualMachineCvesCveTableProps) {
    const { page, perPage } = pagination;

    const fetchVirtualMachineCVEs = useCallback(
        () => listVMCVEs({ searchFilter, page, perPage }),
        [searchFilter, page, perPage]
    );
    const { data, isLoading, error } = useRestQuery(fetchVirtualMachineCVEs);

    const totalCount = data?.totalCount ?? 0;

    const tableState = getTableUIState({
        isLoading,
        data: data?.cves ?? [],
        error,
        searchFilter,
    });

    return (
        <>
            <TableEntityToolbar
                filterToolbar={<VirtualMachineCvesFilterToolbar />}
                entityToggleGroup={<VirtualMachineCvesToggleGroup />}
                pagination={pagination}
                tableRowCount={totalCount}
                isFiltered={isFiltered}
            />
            <Table
                borders={tableState.type === 'COMPLETE'}
                variant="compact"
                aria-live="polite"
                aria-busy={isLoading}
            >
                <Thead noWrap>
                    <Tr>
                        <Th>CVE</Th>
                        <Th>Virtual machines by severity</Th>
                        <Th>Top CVSS</Th>
                        <Th>Affected virtual machines</Th>
                        <Th>EPSS probability</Th>
                        <Th>First discovered</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={6}
                    emptyProps={{
                        message:
                            'No CVEs have been detected for virtual machines across your secured clusters',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((virtualMachineCve) => {
                                const severityCounts = virtualMachineCve.vmSeverityCounts;
                                return (
                                    <Tr key={virtualMachineCve.cve}>
                                        <Td dataLabel="CVE" modifier="nowrap">
                                            <Link
                                                to={getVirtualMachineEntityPagePath(
                                                    'CVE',
                                                    virtualMachineCve.cve
                                                )}
                                            >
                                                {virtualMachineCve.cve}
                                            </Link>
                                        </Td>
                                        <Td dataLabel="Virtual machines by severity">
                                            <SeverityCountLabels
                                                criticalCount={severityCounts?.critical?.total ?? 0}
                                                importantCount={
                                                    severityCounts?.important?.total ?? 0
                                                }
                                                moderateCount={severityCounts?.moderate?.total ?? 0}
                                                lowCount={severityCounts?.low?.total ?? 0}
                                                unknownCount={severityCounts?.unknown?.total ?? 0}
                                                entity="virtual machine"
                                            />
                                        </Td>
                                        <Td dataLabel="Top CVSS">
                                            <CvssFormatted
                                                cvss={virtualMachineCve.topCvss}
                                                scoreVersion={virtualMachineCve.cvssVersion}
                                            />
                                        </Td>
                                        <Td dataLabel="Affected virtual machines">
                                            {`${virtualMachineCve.affectedVmCount} / ${virtualMachineCve.totalVmCount} affected VMs`}
                                        </Td>
                                        <Td dataLabel="EPSS probability">
                                            {formatEpssProbabilityAsPercent(
                                                virtualMachineCve.epssProbability
                                            )}
                                        </Td>
                                        <Td dataLabel="First discovered">
                                            <DateDistance date={virtualMachineCve.publishedOn} />
                                        </Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
        </>
    );
}

export default VirtualMachineCvesCveTable;
