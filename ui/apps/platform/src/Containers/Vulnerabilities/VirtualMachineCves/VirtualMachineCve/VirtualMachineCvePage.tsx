import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import { getVMCVEDetail } from 'services/VirtualMachineService';

import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import { getOverviewPagePath } from '../../utils/searchUtils';
import AffectedVirtualMachinesSummaryCard from './AffectedVirtualMachinesSummaryCard';
import VirtualMachineCvePageHeader from './VirtualMachineCvePageHeader';

const virtualMachineCveOverviewCvePath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'CVE',
});

function VirtualMachineCvePage() {
    const { cveId } = useParams<{ cveId: string }>();

    const fetchCveDetail = useCallback(() => getVMCVEDetail(cveId ?? ''), [cveId]);
    const { data: cveDetail, isLoading, error } = useRestQuery(fetchCveDetail);

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
            <PageSection>
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
            </PageSection>
        </>
    );
}

export default VirtualMachineCvePage;
