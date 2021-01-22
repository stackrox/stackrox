import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { workflowPaths } from 'routePaths';
import PageNotFound from 'Components/PageNotFound';
import isEqual from 'lodash/isEqual';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import Dashboard from './Dashboard/Page';
import Entity from './Entity/Page';
import List from './List/Page';

const Page = () => (
    <searchContext.Provider value={searchParams.page}>
        <Switch>
            <Route exact path={workflowPaths.DASHBOARD} component={Dashboard} />
            <Route path={workflowPaths.LIST} component={List} />
            <Route path={workflowPaths.ENTITY} component={Entity} />
            <Route render={PageNotFound} />
        </Switch>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
