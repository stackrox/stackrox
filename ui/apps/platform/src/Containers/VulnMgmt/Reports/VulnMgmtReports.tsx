import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { vulnManagementReportsPath, vulnManagementReportsPathWithParam } from 'routePaths';
import VulnMgmtReportsMainPage from './VulnMgmtReportsMainPage';
import VulnMgmtReportPage from './VulnMgmtReportPage';

function VulnMgmtReports(): ReactElement {
    return (
        <Switch>
            <Route exact path={vulnManagementReportsPath} component={VulnMgmtReportsMainPage} />
            <Route path={vulnManagementReportsPathWithParam} component={VulnMgmtReportPage} />
        </Switch>
    );
}

export default VulnMgmtReports;
