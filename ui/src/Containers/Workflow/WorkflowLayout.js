import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { nestedPaths as PATHS } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import searchContexts from 'constants/searchContexts';
import DashboardPage from './WorkflowDashboardLayout';
import ListPage from './WorkflowListPageLayout';
import EntityPage from './WorkflowEntityPageLayout';

const Page = React.memo(
    () => (
        <searchContext.Provider value={searchContexts.page}>
            <Switch>
                <Route exact path={PATHS.DASHBOARD} component={DashboardPage} />
                <Route path={PATHS.ENTITY} component={EntityPage} />
                <Route path={PATHS.LIST} component={ListPage} />
                <Route render={PageNotFound} />
            </Switch>
        </searchContext.Provider>
    ),
    isEqual
);

export default Page;
