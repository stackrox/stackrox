import { Route, Routes } from 'react-router-dom-v5-compat';
import { Bullseye, PageSection } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import VirtualMachineCvesOverviewPage from './Overview/VirtualMachineCvesOverviewPage';
import VirtualMachinePage from './VirtualMachine/VirtualMachinePage';

function VirtualMachineCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );

    if (isEnhancedDataModelEnabled) {
        return (
            <PageSection>
                <PageTitle title="Virtual Machine CVEs" />
                <Bullseye>
                    <EmptyStateTemplate
                        title="Enhanced data model preview is active"
                        headingLevel="h1"
                        icon={ExclamationTriangleIcon}
                        status="warning"
                    >
                        The enhanced virtual machine data model is currently enabled. This preview
                        feature only supports API access. The UI for virtual machine CVEs is not
                        available while this feature is active.
                    </EmptyStateTemplate>
                </Bullseye>
            </PageSection>
        );
    }

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Routes>
                <Route index element={<VirtualMachineCvesOverviewPage />} />
                <Route path="virtualmachines/:virtualMachineId" element={<VirtualMachinePage />} />
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
