import { useCallback } from 'react';
import { Flex, Pagination } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import { listVMCVEs } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import SeverityCountLabels from '../../components/SeverityCountLabels';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { formatEpssProbabilityAsPercent } from '../../WorkloadCves/Tables/table.utils';

function VirtualMachineCVEsTable() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const fetchVirtualMachineCVEs = useCallback(
        () => listVMCVEs({ page, perPage }),
        [page, perPage]
    );
    const { data, isLoading, error } = useRestQuery(fetchVirtualMachineCVEs);

    const tableState = getTableUIState({
        isLoading,
        data: data?.cves ?? [],
        error,
        searchFilter: {},
    });

    return (
        <>
            <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                <Pagination
                    itemCount={data?.totalCount ?? 0}
                    perPage={perPage}
                    page={page}
                    onSetPage={(_, newPage) => setPage(newPage)}
                    onPerPageSelect={(_, newPerPage) => {
                        setPerPage(newPerPage);
                    }}
                />
            </Flex>
            <Table
                borders={tableState.type === 'COMPLETE'}
                variant="compact"
                aria-live="polite"
                aria-busy={isLoading ? 'true' : 'false'}
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
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((virtualMachineCve) => {
                                const severityCounts = virtualMachineCve.vmSeverityCounts;
                                return (
                                    <Tr key={virtualMachineCve.cve}>
                                        <Td dataLabel="CVE" modifier="nowrap">
                                            {virtualMachineCve.cve}
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

export default VirtualMachineCVEsTable;
