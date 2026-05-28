import { Route, Routes } from 'react-router-dom-v5-compat';
import { Banner } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import ComplianceNotFoundPage from '../ComplianceNotFoundPage';
import DiscoveredScanConfigDetailPage from './DiscoveredScanConfigDetailPage';
import ScanConfigDetailPage from './ScanConfigDetailPage';
import ScanConfigsTablePage from './ScanConfigsTablePage';

function ScanConfigsPage() {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');

    return (
        <>
            <Banner color="blue" className="pf-v6-u-text-align-center">
                This feature is only available for clusters running Compliance Operator v.1.6 or
                newer
            </Banner>
            <Routes>
                <Route
                    index
                    element={
                        <ScanConfigsTablePage
                            hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                        />
                    }
                />
                <Route
                    path="discovered/:scanConfigName"
                    element={<DiscoveredScanConfigDetailPage />}
                />
                <Route
                    path=":scanConfigId"
                    element={
                        <ScanConfigDetailPage
                            hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                        />
                    }
                />
                <Route path="*" element={<ComplianceNotFoundPage />} />
            </Routes>
        </>
    );
}

export default ScanConfigsPage;
