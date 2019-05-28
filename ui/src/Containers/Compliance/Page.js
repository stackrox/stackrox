import React, { Component } from 'react';
import { withRouter, Route, Switch } from 'react-router-dom';
import { nestedCompliancePaths as PATHS } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import Dashboard from './Dashboard/Page';
import ClusterPage from './Entity/Cluster';
import NamespacePage from './Entity/Namespace';
import NodePage from './Entity/Node';
import DeploymentPage from './Entity/Deployment';
import List from './List/Page';
import ControlPage from './Entity/Control';

class Page extends Component {
    shouldComponentUpdate(nextProps) {
        return !isEqual(nextProps, this.props);
    }

    render() {
        return (
            <Switch>
                <Route exact path={PATHS.DASHBOARD} component={Dashboard} />
                <Route exact path={PATHS.CLUSTER} component={ClusterPage} />
                <Route exact path={PATHS.NAMESPACE} component={NamespacePage} />
                <Route exact path={PATHS.NODE} component={NodePage} />
                <Route exact path={PATHS.CONTROL} component={ControlPage} />
                <Route exact path={PATHS.DEPLOYMENT} component={DeploymentPage} />
                <Route exact path={PATHS.LIST} component={List} />
                <Route render={PageNotFound} />
            </Switch>
        );
    }
}

export default withRouter(Page);
