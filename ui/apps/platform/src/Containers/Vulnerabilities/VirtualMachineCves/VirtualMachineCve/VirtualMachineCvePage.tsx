import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    PageSection,
    Pagination,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getVMCVEDetail, listVMCVEAffectedVMs } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import {
    virtualMachineComponentSearchFilterConfig,
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
} from '../../searchFilterConfig';
import {
    getHiddenSeverities,
    getOverviewPagePath,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import AffectedVirtualMachinesSummaryCard from './AffectedVirtualMachinesSummaryCard';
import AffectedVirtualMachinesTable, {
    defaultSortOption,
    sortFields,
} from './AffectedVirtualMachinesTable';
import VirtualMachineCvePageHeader from './VirtualMachineCvePageHeader';

const searchFilterConfig = [
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
];

const virtualMachineCveOverviewCvePath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'CVE',
});

function VirtualMachineCvePage() {
    const { cveId } = useParams<{ cveId: string }>();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({ sortFields, defaultSortOption });

    const fetchCveDetail = useCallback(
        () => getVMCVEDetail(cveId ?? '', parseQuerySearchFilter(searchFilter)),
        [cveId, searchFilter]
    );
    const { data: cveDetail, isLoading, error } = useRestQuery(fetchCveDetail);

    const fetchAffectedVMs = useCallback(
        () =>
            listVMCVEAffectedVMs(cveId ?? '', {
                searchFilter: parseQuerySearchFilter(searchFilter),
                sortOption,
                page,
                perPage,
            }),
        [cveId, searchFilter, sortOption, page, perPage]
    );
    const {
        data: affectedVMsData,
        isLoading: isLoadingAffectedVMs,
        error: affectedVMsError,
    } = useRestQuery(fetchAffectedVMs);

    const tableState = getTableUIState({
        isLoading: isLoadingAffectedVMs,
        data: affectedVMsData?.vms ?? [],
        error: affectedVMsError,
        searchFilter,
    });

    const affectedVMCount = affectedVMsData?.totalCount ?? 0;

    return (
        <>
            <PageTitle title={`Virtual Machine CVEs - ${cveId}`} />
            <PageSection>
                <Breadcrumb>
                    <BreadcrumbItemLink to={virtualMachineCveOverviewCvePath}>
                        CVEs
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{cveId}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <VirtualMachineCvePageHeader cveDetail={cveDetail} />
            </PageSection>
            <Divider component="div" />
            <PageSection hasBodyWrapper={false}>
                <AdvancedFiltersToolbar
                    searchFilterConfig={searchFilterConfig}
                    searchFilter={searchFilter}
                    defaultSearchFilterEntity="Virtual machine"
                    additionalContextFilter={{ CVE: `"${cveId ?? ''}"` }}
                    onFilterChange={(newFilter) => {
                        setSearchFilter(newFilter);
                        setPage(1);
                    }}
                />
                <SummaryCardLayout error={error} isLoading={isLoading}>
                    <SummaryCard
                        data={cveDetail}
                        loadingText="Loading affected virtual machines summary"
                        renderer={({ data }) => (
                            <AffectedVirtualMachinesSummaryCard
                                affectedVirtualMachinesCount={data.affectedVmCount}
                                totalVirtualMachinesCount={data.totalVmCount}
                                affectedGuestOsCount={data.affectedGuestOsCount}
                            />
                        )}
                    />
                    <SummaryCard
                        data={cveDetail}
                        loadingText="Loading virtual machines by CVE severity summary"
                        renderer={({ data }) => (
                            <BySeveritySummaryCard
                                title="VMs by severity"
                                severityCounts={data.vmSeverityCounts}
                                hiddenSeverities={getHiddenSeverities(querySearchFilter)}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                    <Pagination
                        itemCount={affectedVMCount}
                        perPage={perPage}
                        page={page}
                        onSetPage={(_, newPage) => setPage(newPage)}
                        onPerPageSelect={(_, newPerPage) => {
                            setPerPage(newPerPage);
                        }}
                    />
                </Flex>
                <AffectedVirtualMachinesTable
                    cveId={cveId ?? ''}
                    tableState={tableState}
                    getSortParams={getSortParams}
                    onClearFilters={() => {
                        setSearchFilter({});
                        setPage(1);
                    }}
                />
            </PageSection>
        </>
    );
}

export default VirtualMachineCvePage;
