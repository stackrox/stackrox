import React from 'react';
import { Route, Switch } from 'react-router-dom';

import { complianceEnhancedStatusPath } from 'routePaths';
import ComplianceDashboardPage from './Dashboard/ComplianceDashboardPage';

function ComplianceStatusPage() {
    return (
        <>
            <Switch>
                <Route
                    exact
                    path={complianceEnhancedStatusPath}
                    component={ComplianceDashboardPage}
                />
            </Switch>
        </>
    );
}

export default ComplianceStatusPage;
