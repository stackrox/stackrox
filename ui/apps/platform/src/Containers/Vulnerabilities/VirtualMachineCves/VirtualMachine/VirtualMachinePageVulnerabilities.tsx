import { useCallback } from 'react';
import { Flex, FlexItem, Pagination, Truncate } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import pluralize from 'pluralize';

import CvssFormatted from 'Components/CvssFormatted';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useSet from 'hooks/useSet';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { listVMCVEsByVM } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import {
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
} from '../../searchFilterConfig';
import {
    CVE_EPSS_PROBABILITY_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import { formatEpssProbabilityAsPercent } from '../../WorkloadCves/Tables/table.utils';

const sortFields = [
    CVE_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
    CVE_EPSS_PROBABILITY_SORT_FIELD,
];

const defaultSortOption = { field: CVE_SEVERITY_SORT_FIELD, direction: 'desc' } as const;

const searchFilterConfig = [
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
];

export type VirtualMachinePageVulnerabilitiesProps = {
    virtualMachineId: string;
};

function VirtualMachinePageVulnerabilities({
    virtualMachineId,
}: VirtualMachinePageVulnerabilitiesProps) {
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });
    const expandedRowSet = useSet<string>();
    const colSpan = 7;

    const fetchCVEs = useCallback(
        () => listVMCVEsByVM(virtualMachineId, { searchFilter, page, perPage, sortOption }),
        [virtualMachineId, searchFilter, page, perPage, sortOption]
    );
    const { data, isLoading, error } = useRestQuery(fetchCVEs);

    const tableState = getTableUIState({
        isLoading,
        data: data?.cves ?? [],
        error,
        searchFilter,
    });

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    return (
        <Flex direction={{ default: 'column' }} gap={{ default: 'gapMd' }}>
            <VirtualMachineScanScopeAlert />
            <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                <FlexItem fullWidth={{ default: 'fullWidth' }}>
                    <AdvancedFiltersToolbar
                        className="pf-v6-u-px-sm pf-v6-u-pb-0"
                        defaultSearchFilterEntity="CVE"
                        searchFilter={searchFilter}
                        searchFilterConfig={searchFilterConfig}
                        onFilterChange={(newFilter) => {
                            setSearchFilter(newFilter);
                            setPage(1, 'replace');
                        }}
                    />
                </FlexItem>
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
            <Table borders={tableState.type === 'COMPLETE'} variant="compact" aria-live="polite">
                <Thead noWrap>
                    <Tr>
                        <Th screenReaderText="Row expansion" />
                        <Th sort={getSortParams(CVE_SORT_FIELD)}>CVE</Th>
                        <Th sort={getSortParams(CVE_SEVERITY_SORT_FIELD)}>Top CVE severity</Th>
                        <Th sort={getSortParams(CVE_STATUS_SORT_FIELD)}>CVE status</Th>
                        <Th sort={getSortParams(CVSS_SORT_FIELD)}>Top CVSS</Th>
                        <Th sort={getSortParams(CVE_EPSS_PROBABILITY_SORT_FIELD)}>
                            EPSS probability
                        </Th>
                        <Th>Affected components</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={colSpan}
                    emptyProps={{
                        message: 'No CVEs were detected for this virtual machine',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) =>
                        data.map((cve, rowIndex) => {
                            const isExpanded = expandedRowSet.has(cve.cve);
                            return (
                                <Tbody key={cve.cve} isExpanded={isExpanded}>
                                    <Tr>
                                        <Td
                                            expand={{
                                                rowIndex,
                                                isExpanded,
                                                onToggle: () => expandedRowSet.toggle(cve.cve),
                                            }}
                                        />
                                        <Td dataLabel="CVE">
                                            <Truncate position="middle" content={cve.cve} />
                                        </Td>
                                        <Td dataLabel="Top CVE severity" modifier="nowrap">
                                            <VulnerabilitySeverityIconText
                                                severity={cve.severity}
                                            />
                                        </Td>
                                        <Td dataLabel="CVE status" modifier="nowrap">
                                            <VulnerabilityFixableIconText
                                                isFixable={cve.isFixable}
                                            />
                                        </Td>
                                        <Td dataLabel="Top CVSS" modifier="nowrap">
                                            <CvssFormatted cvss={cve.cvss} />
                                        </Td>
                                        <Td dataLabel="EPSS probability">
                                            {formatEpssProbabilityAsPercent(cve.epssProbability)}
                                        </Td>
                                        <Td dataLabel="Affected components">
                                            {`${cve.affectedComponentCount} ${pluralize('component', cve.affectedComponentCount)}`}
                                        </Td>
                                    </Tr>
                                    <Tr isExpanded={isExpanded}>
                                        <Td />
                                        <Td colSpan={colSpan - 1}>
                                            <ExpandableRowContent>
                                                Affected component details coming soon
                                            </ExpandableRowContent>
                                        </Td>
                                    </Tr>
                                </Tbody>
                            );
                        })
                    }
                />
            </Table>
        </Flex>
    );
}

export default VirtualMachinePageVulnerabilities;
