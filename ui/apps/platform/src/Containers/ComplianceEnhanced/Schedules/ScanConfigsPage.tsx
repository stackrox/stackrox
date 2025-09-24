import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom-v5-compat';
import { Banner } from '@patternfly/react-core';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';
import { complianceEnhancedSchedulesPath } from 'routePaths';
import type { PageActions } from './compliance.scanConfigs.utils';
import CreateScanConfigPage from './CreateScanConfigPage';
import ComplianceNotFoundPage from '../ComplianceNotFoundPage';
import ScanConfigDetailPage from './ScanConfigDetailPage';
import ScanConfigsTablePage from './ScanConfigsTablePage';

function ScanConfigsPage() {
    /*
     * Examples of urls for ScanConfigPage:
     * /main/compliance-enhanced/scan-configs
     * /main/compliance-enhanced/scan-configs?action=create
     * /main/compliance-enhanced/scan-configs/configId
     */
    const { pageAction } = usePageAction<PageActions>();

    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');

    return (
        <>
            <Banner variant="blue" className="pf-v5-u-text-align-center">
                This feature is only available for clusters running Compliance Operator v.1.6 or
                newer
            </Banner>
            <Routes>
                <Route
                    index
                    element={
                        pageAction === 'create' && hasWriteAccessForCompliance ? (
                            <CreateScanConfigPage />
                        ) : !pageAction ? (
                            <ScanConfigsTablePage
                                hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                            />
                        ) : (
                            <Navigate to={complianceEnhancedSchedulesPath} replace />
                        )
                    }
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
