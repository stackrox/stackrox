import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportsPath } from 'routePaths';

import VulnReportsPage from './VulnReports/VulnReportsPage';
import CreateVulnReportPage from './CreateVulnReport/CreateVulnReportPage';

import './VulnReportingPage.css';
import usePageAction from './hooks/usePageState';

type PageActions = 'create' | 'edit' | 'clone';

function VulnReportingPage() {
    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const { pageAction } = usePageAction<PageActions>();

    const hasWorkflowAdministrationWriteAccess = hasReadWriteAccess('WorkflowAdministration');
    const hasImageReadAccess = hasReadAccess('Image');
    const hasAccessScopeReadAccess = hasReadAccess('Access');
    const hasNotifierIntegrationReadAccess = hasReadAccess('Integration');
    const canReadWriteReports =
        hasWorkflowAdministrationWriteAccess &&
        hasImageReadAccess &&
        hasAccessScopeReadAccess &&
        hasNotifierIntegrationReadAccess;

    return (
        <Switch>
            <Route
                exact
                path={vulnerabilityReportsPath}
                render={(props) => {
                    if (pageAction === 'create' && canReadWriteReports) {
                        return <CreateVulnReportPage {...props} />;
                    }
                    if (pageAction === undefined) {
                        return <VulnReportsPage {...props} />;
                    }
                    return <Redirect to={vulnerabilityReportsPath} />;
                }}
            />
        </Switch>
    );
}

export default VulnReportingPage;
