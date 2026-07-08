import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import { getVMCVEDetail } from 'services/VirtualMachineService';

import { getOverviewPagePath } from '../../utils/searchUtils';
import VirtualMachineCvePageHeader from './VirtualMachineCvePageHeader';

const virtualMachineCveOverviewCvePath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'CVE',
});

function VirtualMachineCvePage() {
    const { cveId } = useParams<{ cveId: string }>();

    const fetchCveDetail = useCallback(() => getVMCVEDetail(cveId ?? ''), [cveId]);
    const { data: cveDetail } = useRestQuery(fetchCveDetail);

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
        </>
    );
}

export default VirtualMachineCvePage;
