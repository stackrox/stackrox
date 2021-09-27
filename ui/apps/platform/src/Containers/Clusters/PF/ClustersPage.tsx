import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { clustersListPath } from 'routePaths';

import ClustersNotFoundPage from './ClustersNotFoundPage';
import ClustersListPage from './ClustersListPage';

const ClustersPage = (): ReactElement => (
    <Switch>
        <Route exact path={clustersListPath} component={ClustersListPage} />
        <Route component={ClustersNotFoundPage} />
    </Switch>
);

export default ClustersPage;
