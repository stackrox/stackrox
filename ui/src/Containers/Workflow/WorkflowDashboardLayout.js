import React from 'react';
import URLService from 'modules/URLService';
import useCaseTypes from 'constants/useCaseTypes';

import VulnMgmtDashboardPage from 'Containers/VulnMgmt/Dashboard/VulnMgmtDashboardPage';

const DashboardMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtDashboardPage
};

const WorkflowDashboardLayout = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const { context: useCase } = params;
    const Dashboard = DashboardMap[useCase];

    return <Dashboard {...params} />;
};

export default WorkflowDashboardLayout;
