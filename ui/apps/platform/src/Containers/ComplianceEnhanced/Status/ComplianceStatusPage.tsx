import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    complianceEnhancedCoverageClustersPath,
    complianceEnhancedStatusPath,
    complianceEnhancedCoverageProfilesPath,
    complianceEnhancedStatusScansPath,
} from 'routePaths';

import ComplianceDashboardPage from './Dashboard/ComplianceDashboardPage';
import ComplianceReportsClusterPage from './Reports/ComplianceReportsClusterPage';
import ComplianceReportsProfilePage from './Reports/ComplianceReportsProfilePage';
import ComplianceReportsScanPage from './Reports/ComplianceReportsScanPage';

function ComplianceStatusPage() {
    return (
        <>
            <Switch>
                <Route
                    path={complianceEnhancedCoverageClustersPath}
                    component={ComplianceReportsClusterPage}
                />
                <Route
                    path={complianceEnhancedCoverageProfilesPath}
                    component={ComplianceReportsProfilePage}
                />
                <Route
                    path={complianceEnhancedStatusScansPath}
                    component={ComplianceReportsScanPage}
                />
                <Route
                    exact
                    path={complianceEnhancedStatusPath}
                    component={ComplianceDashboardPage}
                />
                <Route
                    path={`${complianceEnhancedStatusPath}/*`}
                    render={() => <Redirect to={complianceEnhancedStatusPath} />}
                />
            </Switch>
        </>
    );
}

export default ComplianceStatusPage;
