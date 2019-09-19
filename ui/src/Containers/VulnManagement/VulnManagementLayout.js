import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { nestedPaths as PATHS } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import VulnDashboardPage from './VulnDashboard/VulnDashboardPage';

const VulnManagementLayout = React.memo(
    () => (
        <Switch>
            <Route exact path={PATHS.DASHBOARD} component={VulnDashboardPage} />
            <Route render={PageNotFound} />
        </Switch>
    ),
    isEqual
);

export default VulnManagementLayout;
