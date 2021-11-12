import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { vulnManagementReportingPath } from 'routePaths';
import VulnMgmtReportTablePage from './VulnMgmtReportTablePage';

function ReportPage(): ReactElement {
    return (
        <Switch>
            <Route exact path={vulnManagementReportingPath} component={VulnMgmtReportTablePage} />
        </Switch>
    );
}

export default ReportPage;
