import React, { Component } from 'react';
import { Route, Switch } from 'react-router-dom';
import { nestedPaths as PATHS } from 'routePaths';
import isEqual from 'lodash/isEqual';

import PageNotFound from 'Components/PageNotFound';
import DashboardPage from './Dashboard/Page';
import ListPage from './List/Page';
import EntityPage from './Entity/Page';

class Page extends Component {
    shouldComponentUpdate(nextProps) {
        return !isEqual(nextProps, this.props);
    }

    render() {
        return (
            <Switch>
                <Route exact path={PATHS.DASHBOARD} component={DashboardPage} />
                <Route path={PATHS.ENTITY} component={EntityPage} />
                <Route path={PATHS.LIST} component={ListPage} />
                <Route render={PageNotFound} />
            </Switch>
        );
    }
}

export default Page;
