import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { Banner } from '@patternfly/react-core';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';
import { complianceEnhancedSchedulesPath } from 'routePaths';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';
import { PageActions } from './compliance.scanConfigs.utils';
import CreateScanConfigPage from './CreateScanConfigPage';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isReportJobsEnabled = isFeatureFlagEnabled('ROX_SCAN_SCHEDULE_REPORT_JOBS');

    return (
        <>
            {isReportJobsEnabled && (
                <Banner variant="blue" className="pf-v5-u-text-align-center">
                    Reporting is only available for clusters running Compliance Operator v.1.6 or
                    newer
                </Banner>
            )}
            <Switch>
                <Route
                    exact
                    path={complianceEnhancedSchedulesPath}
                    render={() => {
                        if (pageAction === 'create' && hasWriteAccessForCompliance) {
                            return <CreateScanConfigPage />;
                        }
                        if (!pageAction) {
                            return (
                                <ScanConfigsTablePage
                                    hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                                    isReportJobsEnabled={isReportJobsEnabled}
                                />
                            );
                        }
                        return <Redirect to={complianceEnhancedSchedulesPath} />;
                    }}
                />
                <Route
                    exact
                    path={scanConfigDetailsPath}
                    render={() => {
                        return (
                            <ScanConfigDetailPage
                                hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                                isReportJobsEnabled={isReportJobsEnabled}
                            />
                        );
                    }}
                />
            </Switch>
        </>
    );
}

export default ScanConfigsPage;
