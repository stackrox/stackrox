import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    complianceEnhancedBasePath,
    complianceEnhancedClusterComplianceBasePath,
    complianceEnhancedStatusPath,
} from 'routePaths';
import ComplianceStatusPage from './Status/ComplianceStatusPage';
import ClusterCompliancePage from './ClusterCompliance/ClusterCompliancePage';

function ComplianceEnhancedPage() {
    return (
        <Switch>
            <Redirect exact from={complianceEnhancedBasePath} to={complianceEnhancedStatusPath} />
            <Route path={complianceEnhancedStatusPath}>
                <ComplianceStatusPage />
            </Route>
            <Route path={complianceEnhancedClusterComplianceBasePath}>
                <ClusterCompliancePage />
            </Route>
        </Switch>
    );
}

export default ComplianceEnhancedPage;
