import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { vulnManagementReportsPath, vulnManagementReportsPathWithParam } from 'routePaths';
import VulnMgmtReportsBasePage from './VulnMgmtReportsBasePage';
import VulnMgmtReportSinglePage from './VulnMgmtReportSinglePage';

function VulnMgmtReports(): ReactElement {
    return (
        <Switch>
            <Route exact path={vulnManagementReportsPath} component={VulnMgmtReportsBasePage} />
            <Route path={vulnManagementReportsPathWithParam} component={VulnMgmtReportSinglePage} />
        </Switch>
    );
}

export default VulnMgmtReports;
