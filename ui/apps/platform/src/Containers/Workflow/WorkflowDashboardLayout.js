import React from 'react';
import URLService from 'utils/URLService';
import useCaseTypes from 'constants/useCaseTypes';
import { withRouter } from 'react-router-dom';
import parseURL from 'utils/URLParser';
import workflowStateContext from 'Containers/workflowStateContext';

import VulnMgmtDashboardPage from 'Containers/VulnMgmt/Dashboard/VulnMgmtDashboardPage';

const DashboardMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtDashboardPage,
};

const WorkflowDashboardLayout = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const workflowState = parseURL(location);
    const { context: useCase } = params;
    const Dashboard = DashboardMap[useCase];
    return (
        <workflowStateContext.Provider value={workflowState}>
            <Dashboard {...params} />
        </workflowStateContext.Provider>
    );
};

export default withRouter(WorkflowDashboardLayout);
