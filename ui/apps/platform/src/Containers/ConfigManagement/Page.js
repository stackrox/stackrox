import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { workflowPaths } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import DashboardPage from './Dashboard/Page';
import ListPage from './List/Page';
import EntityPage from './Entity/Page';

const Page = () => (
    <searchContext.Provider value={searchParams.page}>
        <Switch>
            <Route exact path={workflowPaths.DASHBOARD} component={DashboardPage} />
            <Route path={workflowPaths.ENTITY} component={EntityPage} />
            <Route path={workflowPaths.LIST} component={ListPage} />
            <Route render={PageNotFound} />
        </Switch>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
