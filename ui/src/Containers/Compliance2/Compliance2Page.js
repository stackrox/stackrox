import React from 'react';
import { withRouter, Route } from 'react-router-dom';
import { nestedCompliancePaths as PATHS } from 'routePaths';
import Dashboard from './ComplianceDashboard';
import Entity from './ComplianceEntity';
import List from './ComplianceList';

const Page = () => (
    <>
        <Route exact path={PATHS.DASHBOARD} component={Dashboard} />
        <Route exact path={PATHS.RESOURCE} component={Entity} />
        <Route exact path={PATHS.CONTROL} component={Entity} />
        <Route exact path={PATHS.LIST} component={List} />
    </>
);

export default withRouter(Page);
