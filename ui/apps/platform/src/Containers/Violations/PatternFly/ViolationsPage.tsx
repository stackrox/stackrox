import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { violationsPFBasePath, violationsPFPath } from 'routePaths';
import ViolationsTablePage from './ViolationsTablePage';
import ViolationDetailsPage from './Details/ViolationDetailsPage';
import ViolationNotFoundPage from './ViolationNotFoundPage';

function ViolationsPage(): ReactElement {
    return (
        <Switch>
            <Route exact path={violationsPFBasePath} component={ViolationsTablePage} />
            <Route path={violationsPFPath} component={ViolationDetailsPage} />
            <Route component={ViolationNotFoundPage} />
        </Switch>
    );
}

export default ViolationsPage;
