import React from 'react';
import { Route, Switch, withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { nestedPaths as PATHS } from 'routePaths';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import workflowStateContext from 'Containers/workflowStateContext';
import { parseURL } from 'modules/URLReadWrite';
import searchContexts from 'constants/searchContexts';
import DashboardPage from './WorkflowDashboardLayout';
import ListPage from './WorkflowListPageLayout';
import EntityPage from './WorkflowEntityPageLayout';

const Page = ({ location }) => {
    const { workflowState } = parseURL(location);
    return (
        <workflowStateContext.Provider value={workflowState}>
            <searchContext.Provider value={searchContexts.page}>
                <Switch>
                    <Route exact path={PATHS.DASHBOARD} component={DashboardPage} />
                    <Route path={PATHS.ENTITY} component={EntityPage} />
                    <Route path={PATHS.LIST} component={ListPage} />
                    <Route render={PageNotFound} />
                </Switch>
            </searchContext.Provider>
        </workflowStateContext.Provider>
    );
};

Page.propTypes = {
    location: ReactRouterPropTypes.location
};

Page.defaultProps = {
    location: null
};

export default withRouter(React.memo(Page, isEqual));
