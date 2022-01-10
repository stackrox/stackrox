import React, { ReactElement } from 'react';
import { Route, Switch, useHistory } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';
import {
    vulnManagementReportsPath,
    vulnManagementReportsPathWithParam,
    dashboardPath,
} from 'routePaths';
import VulnMgmtReportsBasePage from './VulnMgmtReportsBasePage';
import VulnMgmtReportSinglePage from './VulnMgmtReportSinglePage';

function VulnMgmtReports(): ReactElement {
    const history = useHistory();

    const { hasReadAccess } = usePermissions();
    const hasVulnReportReadAccess = hasReadAccess('VulnerabilityReports');

    if (!hasVulnReportReadAccess) {
        history.replace(dashboardPath);
    }

    return (
        <Switch>
            <Route exact path={vulnManagementReportsPath} component={VulnMgmtReportsBasePage} />
            <Route path={vulnManagementReportsPathWithParam} component={VulnMgmtReportSinglePage} />
        </Switch>
    );
}

export default VulnMgmtReports;
