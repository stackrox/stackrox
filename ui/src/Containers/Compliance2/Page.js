import React from 'react';
import { withRouter, Route } from 'react-router-dom';
import { nestedCompliancePaths as PATHS } from 'routePaths';
import DashboardPage from './Dashboard/Page';
import EntityPage from './Entity';
import ListPage from './List/Page';

const CompliancePage = () => (
    <>
        <Route exact path={PATHS.DASHBOARD} component={DashboardPage} />
        <Route exact path={PATHS.RESOURCE} component={EntityPage} />
        <Route exact path={PATHS.CONTROL} component={EntityPage} />
        <Route exact path={PATHS.LIST} component={ListPage} />
    </>
);

export default withRouter(CompliancePage);
