import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { workflowPaths } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import useCases from 'constants/useCaseTypes';
import DashboardPage from './Dashboard/Page';
import ListPage from './List/Page';
import EntityPage from './Entity/Page';

const Page = () => (
    <searchContext.Provider value={searchParams.page}>
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
                <PageNotFound useCase={useCases.CONFIG_MANAGEMENT} />
            </Route>
        </Switch>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
