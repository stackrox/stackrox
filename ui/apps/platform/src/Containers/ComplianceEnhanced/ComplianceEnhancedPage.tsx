import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    complianceEnhancedStatusPath,
    complianceEnhancedBasePath,
    complianceEnhancedScanConfigsPath,
} from 'routePaths';
import ComplianceStatusPage from 'Containers/ComplianceEnhanced/Status/ComplianceStatusPage';
import ScanConfigsPage from 'Containers/ComplianceEnhanced/ScanConfigs/ScanConfigsPage';

function ComplianceEnhancedPage() {
    return (
        <Switch>
            <Redirect exact from={complianceEnhancedBasePath} to={complianceEnhancedStatusPath} />
            <Route path={complianceEnhancedStatusPath}>
                <ComplianceStatusPage />
            </Route>
            <Route path={complianceEnhancedScanConfigsPath}>
                <ScanConfigsPage />
            </Route>
        </Switch>
    );
}

export default ComplianceEnhancedPage;
