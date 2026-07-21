import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    PageSection,
    Pagination,
    Split,
    SplitItem,
    Title,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import { getVMCVEDetail, listVMCVEAffectedVMs } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { getOverviewPagePath } from '../../utils/searchUtils';
import AffectedVirtualMachinesSummaryCard from './AffectedVirtualMachinesSummaryCard';
import AffectedVirtualMachinesTable from './AffectedVirtualMachinesTable';
import VirtualMachineCvePageHeader from './VirtualMachineCvePageHeader';

const virtualMachineCveOverviewCvePath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'CVE',
});

function VirtualMachineCvePage() {
    const { cveId } = useParams<{ cveId: string }>();

    const fetchCveDetail = useCallback(() => getVMCVEDetail(cveId ?? ''), [cveId]);
    const { data: cveDetail, isLoading, error } = useRestQuery(fetchCveDetail);

    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const fetchAffectedVMs = useCallback(
        () => listVMCVEAffectedVMs(cveId ?? '', { page, perPage }),
        [cveId, page, perPage]
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
        searchFilter: {},
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
                                hiddenSeverities={new Set()}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <Split hasGutter className="pf-v6-u-align-items-baseline">
                    <SplitItem isFilled>
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                            <Title headingLevel="h2">
                                {`${affectedVMCount} ${pluralize('virtual machine', affectedVMCount)} affected`}
                            </Title>
                        </Flex>
                    </SplitItem>
                    <SplitItem>
                        <Pagination
                            itemCount={affectedVMCount}
                            perPage={perPage}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </SplitItem>
                </Split>
                <AffectedVirtualMachinesTable
                    tableState={tableState}
                    onClearFilters={() => {
                        setPage(1);
                    }}
                />
            </PageSection>
        </>
    );
}

export default VirtualMachineCvePage;
