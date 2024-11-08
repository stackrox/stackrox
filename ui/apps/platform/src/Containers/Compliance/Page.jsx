import React from 'react';
import { Route, Switch } from 'react-router-dom';
import isEqual from 'lodash/isEqual';

import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import { mainPath } from 'routePaths';

import Dashboard from './Dashboard/ComplianceDashboardPage';
import Entity from './Entity/Page';
import List from './List/Page';

const pageEntityListType = 'clusters|controls|deployments|namespaces|nodes';
const pageEntityType = 'cluster|control|deployment|namespace|node|standard';

// mainPath instead of complianceBasePath because URLService requires param context = 'compliance'
const complianceDashboardPath = `${mainPath}/:context`;
const complianceListPath = `${mainPath}/:context/:pageEntityListType(${pageEntityListType})/:entityId1?/:entityType2?/:entityId2?`;
const complianceEntityPath = `${mainPath}/:context/:pageEntityType(${pageEntityType})/:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`;

const Page = () => (
    <searchContext.Provider value="s">
        <Switch>
            <Route exact path={complianceDashboardPath}>
                <Dashboard />
            </Route>
            <Route path={complianceListPath}>
                <List />
            </Route>
            <Route path={complianceEntityPath}>
                <Entity />
            </Route>
            <Route>
                <PageNotFound useCase="compliance" />
            </Route>
        </Switch>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
