import { useParams } from 'react-router-dom-v5-compat';
import { Breadcrumb, BreadcrumbItem, PageSection } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import { getOverviewPagePath } from '../../utils/searchUtils';

const virtualMachineCveOverviewCvePath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'CVE',
});

function VirtualMachineCvePage() {
    const { cveId } = useParams<{ cveId: string }>();

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
        </>
    );
}

export default VirtualMachineCvePage;
