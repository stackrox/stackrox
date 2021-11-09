import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { vulnManagementReportingPath } from 'routePaths';
import ReportingTablePage from './VulnMgmtReportTablePage';

function ReportingPage(): ReactElement {
    return (
        <Switch>
            <Route exact path={vulnManagementReportingPath} component={ReportingTablePage} />
        </Switch>
    );
}

export default ReportingPage;
