import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';
import {
    complianceEnhancedScanConfigsPath,
    complianceEnhancedScanConfigDetailPath,
} from 'routePaths';

import ScanConfigsTablePage from './Table/ScanConfigsTablePage';
import CreateScanConfigPage from './CreateScanConfigPage';
import ScanConfigDetailPage from './ScanConfigDetailPage';

type PageActions = 'create' | 'edit' | 'clone';

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
        <Switch>
            <Route
                exact
                path={complianceEnhancedScanConfigsPath}
                render={() => {
                    if (pageAction === 'create' && hasWriteAccessForCompliance) {
                        return <CreateScanConfigPage />;
                    }
                    if (pageAction === undefined) {
                        return (
                            <ScanConfigsTablePage
                                hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                            />
                        );
                    }
                    return <Redirect to={complianceEnhancedScanConfigsPath} />;
                }}
            />
            <Route
                exact
                path={complianceEnhancedScanConfigDetailPath}
                render={() => {
                    return (
                        <ScanConfigDetailPage
                            hasWriteAccessForCompliance={hasWriteAccessForCompliance}
                        />
                    );
                }}
            />
        </Switch>
    );
}

export default ScanConfigsPage;
