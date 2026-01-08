import { Route, Routes } from 'react-router-dom-v5-compat';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import { Subnav } from 'Components/Navigation/SubnavContext';
import usePermissions from 'hooks/usePermissions';
import VirtualMachineCvesOverviewPage from './Overview/VirtualMachineCvesOverviewPage';
import VirtualMachinePage from './VirtualMachine/VirtualMachinePage';
import VulnerabilitiesSubnav from '../VulnerabilitiesSubnav';

function VirtualMachineCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    return (
        <>
            <Subnav>
                <VulnerabilitiesSubnav />
            </Subnav>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Routes>
                <Route index element={<VirtualMachineCvesOverviewPage />} />
                <Route path="virtualmachines/:virtualMachineId" element={<VirtualMachinePage />} />
                <Route
                    path="*"
                    element={
                        <PageSection variant="light">
                            <PageTitle title="Virtual Machine CVEs - Not Found" />
                            <PageNotFound />
                        </PageSection>
                    }
                />
            </Routes>
        </>
    );
}

export default VirtualMachineCvesPage;
