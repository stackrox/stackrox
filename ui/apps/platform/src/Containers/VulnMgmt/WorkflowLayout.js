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
        <Route exact path={workflowPaths.DASHBOARD} component={DashboardPage} />
        <Route path={workflowPaths.ENTITY} component={EntityPage} />
        <Route path={workflowPaths.LIST} component={ListPage} />
        <Route render={PageNotFound} />
    </Switch>
);

export default React.memo(Page, isEqual);
