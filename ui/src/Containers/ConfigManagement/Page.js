import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { nestedPaths as PATHS } from 'routePaths';

import PageNotFound from 'Components/PageNotFound';
import DashboardPage from './Dashboard/Page';
import ListPage from './List/Page';
import EntityPage from './Entity/Page';

const Page = () => (
    <Switch>
        <Route exact path={PATHS.DASHBOARD} component={DashboardPage} />
        <Route path={`${PATHS.DASHBOARD}${PATHS.ENTITY}`} component={EntityPage} />
        <Route path={`${PATHS.DASHBOARD}${PATHS.LIST}`} component={ListPage} />
        <Route render={PageNotFound} />
    </Switch>
);

export default Page;
