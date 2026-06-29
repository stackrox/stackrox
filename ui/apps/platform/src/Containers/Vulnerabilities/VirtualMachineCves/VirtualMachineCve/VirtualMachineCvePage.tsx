import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import { getVMCVEDetail } from 'services/VirtualMachineService';

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
            <PageSection hasBodyWrapper={false}>
                <Breadcrumb>
                    <BreadcrumbItemLink to={virtualMachineCveOverviewCvePath}>
                        CVEs
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{cveId}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection hasBodyWrapper={false}>
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
                </SummaryCardLayout>
            </PageSection>
        </>
    );
}

export default VirtualMachineCvePage;
