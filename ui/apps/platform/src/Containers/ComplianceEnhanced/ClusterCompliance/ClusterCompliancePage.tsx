import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    complianceEnhancedClusterComplianceBasePath,
    complianceEnhancedScanConfigsPath,
    complianceEnhancedCoveragePath,
} from 'routePaths';

import CoveragePage from './Coverage/CoveragePage';
import ScanConfigsPage from './ScanConfigs/ScanConfigsPage';

function ClusterCompliancePage() {
    /*
     * Examples of urls for ClusterCompliancePage:
     * /main/compliance-enhanced/cluster-compliance/coverage
     * /main/compliance-enhanced/cluster-compliance/scheduling
     */

    return (
        <Switch>
            <Redirect
                exact
                from={complianceEnhancedClusterComplianceBasePath}
                to={complianceEnhancedCoveragePath}
            />
            <Route path={complianceEnhancedScanConfigsPath}>
                <ScanConfigsPage />
            </Route>
            <Route path={complianceEnhancedCoveragePath}>
                <CoveragePage />
            </Route>
        </Switch>
    );
}

export default ClusterCompliancePage;
