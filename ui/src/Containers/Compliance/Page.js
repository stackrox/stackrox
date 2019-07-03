import React, { Component } from 'react';
import { withRouter, Route, Switch } from 'react-router-dom';
import { nestedPaths as PATHS } from 'routePaths';
import PageNotFound from 'Components/PageNotFound';
import isEqual from 'lodash/isEqual';
import searchContext from 'Containers/searchContext';
import searchContexts from 'constants/searchContexts';
import Dashboard from './Dashboard/Page';
import Entity from './Entity/Page';
import List from './List/Page';

class Page extends Component {
    shouldComponentUpdate(nextProps) {
        return !isEqual(nextProps, this.props);
    }

    render() {
        return (
            <searchContext.Provider value={searchContexts.page}>
                <Switch>
                    <Route exact path={PATHS.DASHBOARD} component={Dashboard} />
                    <Route path={PATHS.LIST} component={List} />
                    <Route path={PATHS.ENTITY} component={Entity} />
                    <Route render={PageNotFound} />
                </Switch>
            </searchContext.Provider>
        );
    }
}

export default withRouter(Page);
