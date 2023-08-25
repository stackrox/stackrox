import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { workflowPaths } from 'routePaths';
import PageNotFound from 'Components/PageNotFound';
import isEqual from 'lodash/isEqual';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import useCases from 'constants/useCaseTypes';
import Dashboard from './Dashboard/ComplianceDashboardPage';
import Entity from './Entity/Page';
import List from './List/Page';

const Page = () => (
    <searchContext.Provider value={searchParams.page}>
        <Switch>
            <Route exact path={workflowPaths.DASHBOARD} component={Dashboard} />
            <Route path={workflowPaths.LIST} component={List} />
            <Route path={workflowPaths.ENTITY} component={Entity} />
            <Route>
                <PageNotFound useCase={useCases.COMPLIANCE} />
            </Route>
        </Switch>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
