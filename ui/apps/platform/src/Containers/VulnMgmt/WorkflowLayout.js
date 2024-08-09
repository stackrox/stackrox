import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { workflowPaths } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';

import DashboardPage from './Dashboard/WorkflowDashboardLayout';
import ListPage from './List/WorkflowListPageLayout';
import EntityPage from './Entity/WorkflowEntityPageLayout';

const Page = () => (
    <Switch>
        <Route exact path={workflowPaths.DASHBOARD}>
            <DashboardPage />
        </Route>
        <Route path={workflowPaths.ENTITY}>
            <EntityPage />
        </Route>
        <Route path={workflowPaths.LIST}>
            <ListPage />
        </Route>
        <Route>
            <PageNotFound />
        </Route>
    </Switch>
);

export default React.memo(Page, isEqual);
