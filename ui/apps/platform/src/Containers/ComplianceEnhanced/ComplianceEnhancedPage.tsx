import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    complianceEnhancedStatusPath,
    complianceEnhancedBasePath,
    complianceEnhancedScanConfigsBasePath,
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
            {/* TODO: see if there is a more elegant solution than 2 Route components for similar paths */}
            <Route path={complianceEnhancedScanConfigsPath}>
                <ScanConfigsPage />
            </Route>
            <Route path={complianceEnhancedScanConfigsBasePath}>
                <ScanConfigsPage />
            </Route>
        </Switch>
    );
}

export default ComplianceEnhancedPage;
