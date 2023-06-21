import React from 'react';
import { Route, Switch } from 'react-router-dom';

import { vulnerabilityReportingPath } from 'routePaths';

import VulnReportsPage from './VulnReports/VulnReportsPage';

import './VulnReportingPage.css';

function VulnReportingPage() {
    return (
        <Switch>
            <Route exact path={vulnerabilityReportingPath} component={VulnReportsPage} />
        </Switch>
    );
}

export default VulnReportingPage;
