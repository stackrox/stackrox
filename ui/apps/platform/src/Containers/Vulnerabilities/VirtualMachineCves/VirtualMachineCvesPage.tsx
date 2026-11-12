import { Route, Routes } from 'react-router-dom-v5-compat';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import useFeatureFlags from 'hooks/useFeatureFlags';
import VirtualMachineCvesOverviewPage from './Overview/VirtualMachineCvesOverviewPage';
import VirtualMachineCvePage from './VirtualMachineCve/VirtualMachineCvePage';
import VirtualMachinePage from './VirtualMachine/VirtualMachinePage';
import VirtualMachinePageLegacy from './VirtualMachine/VirtualMachinePageLegacy';

function VirtualMachineCvesPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );

    return (
        <>
            <Routes>
                <Route index element={<VirtualMachineCvesOverviewPage />} />
                <Route path="cves/:cveId" element={<VirtualMachineCvePage />} />
                <Route
                    path="virtualmachines/:virtualMachineId"
                    element={
                        isEnhancedDataModelEnabled ? (
                            <VirtualMachinePage />
                        ) : (
                            <VirtualMachinePageLegacy />
                        )
                    }
                />
                <Route
                    path="*"
                    element={
                        <PageSection hasBodyWrapper={false}>
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
