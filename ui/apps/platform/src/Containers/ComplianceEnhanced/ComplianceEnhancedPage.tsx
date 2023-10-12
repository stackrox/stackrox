import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    complianceEnhancedStatusPath,
    complianceEnhancedBasePath,
    complianceEnhancedScanConfigsPath,
} from 'routePaths';
import ComplianceStatusPage from 'Containers/ComplianceEnhanced/Status/ComplianceStatusPage';
import SchedulingPage from 'Containers/ComplianceEnhanced/Scheduling/SchedulingPage';

function ComplianceEnhancedPage() {
    return (
        <Switch>
            <Redirect exact from={complianceEnhancedBasePath} to={complianceEnhancedStatusPath} />
            <Route path={complianceEnhancedStatusPath}>
                <ComplianceStatusPage />
            </Route>
            <Route path={complianceEnhancedScanConfigsPath}>
                <SchedulingPage />
            </Route>
        </Switch>
    );
}

export default ComplianceEnhancedPage;
