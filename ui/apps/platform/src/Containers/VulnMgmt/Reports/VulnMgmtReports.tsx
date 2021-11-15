import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { vulnManagementReportsPath } from 'routePaths';
import VulnMgmtReportsMainPage from './VulnMgmtReportsMainPage';

function VulnMgmtReports(): ReactElement {
    return (
        <Switch>
            <Route exact path={vulnManagementReportsPath} component={VulnMgmtReportsMainPage} />
        </Switch>
    );
}

export default VulnMgmtReports;
