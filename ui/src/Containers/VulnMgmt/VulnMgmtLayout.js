import React from 'react';
import { Route, Switch } from 'react-router-dom';

import { nestedPaths as PATHS } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';

import VulnMgmtDashboardPage from './VulnMgmtDashboard/VulnDashboardPage';
import VulnMgmtEntityLayout from './VulnMgmtEntity/VulnMgmtEntityLayout';
import VulnMgmtListLayout from './VulnMgmtList/VulnMgmtListLayout';

const VulnMgmtLayout = React.memo(
    () => (
        <Switch>
            <Route exact path={PATHS.DASHBOARD} component={VulnMgmtDashboardPage} />
            <Route path={PATHS.ENTITY} component={VulnMgmtEntityLayout} />
            <Route path={PATHS.LIST} component={VulnMgmtListLayout} />
            <Route render={PageNotFound} />
        </Switch>
    ),
    isEqual
);

export default VulnMgmtLayout;
