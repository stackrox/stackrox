import React from 'react';
import { withRouter, Route, Switch } from 'react-router-dom';
import { nestedCompliancePaths as PATHS } from 'routePaths';
import Dashboard from './Dashboard/Page';
import Entity from './Entity/Page';
import List from './List/Page';

const Page = () => (
    <Switch>
        <Route exact path={PATHS.DASHBOARD} component={Dashboard} />
        <Route exact path={PATHS.RESOURCE} component={Entity} />
        <Route exact path={PATHS.CONTROL} component={Entity} />
        <Route exact path={PATHS.LIST} component={List} />
    </Switch>
);

export default withRouter(Page);
