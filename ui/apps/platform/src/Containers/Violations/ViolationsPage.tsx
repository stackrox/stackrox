import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { violationsBasePath, violationsPath } from 'routePaths';
import ViolationsTablePage from './ViolationsTablePage';
import ViolationDetailsPage from './Details/ViolationDetailsPage';
import ViolationNotFoundPage from './ViolationNotFoundPage';

function ViolationsPage(): ReactElement {
    return (
        <Switch>
            <Route exact path={violationsBasePath}>
                <ViolationsTablePage />
            </Route>
            <Route path={violationsPath}>
                <ViolationDetailsPage />
            </Route>
            <Route>
                <ViolationNotFoundPage />
            </Route>
        </Switch>
    );
}

export default ViolationsPage;
