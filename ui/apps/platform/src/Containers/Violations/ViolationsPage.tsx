import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { violationsBasePath, violationsPath } from 'routePaths';
import ViolationsTablePage from './ViolationsTablePage';
import ViolationDetailsPage from './Details/ViolationDetailsPage';
import ViolationNotFoundPage from './ViolationNotFoundPage';

function ViolationsPage(): ReactElement {
    return (
        <Switch>
            <Route exact path={violationsBasePath} component={ViolationsTablePage} />
            <Route path={violationsPath} component={ViolationDetailsPage} />
            <Route component={ViolationNotFoundPage} />
        </Switch>
    );
}

export default ViolationsPage;
