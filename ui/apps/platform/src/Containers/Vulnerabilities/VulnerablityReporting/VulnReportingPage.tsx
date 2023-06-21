import React from 'react';
import { Route, Switch } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportingPath, vulnerabilityReportingCreatePath } from 'routePaths';

import VulnReportsPage from './VulnReports/VulnReportsPage';
import CreateVulnReportPage from './CreateVulnReport/CreateVulnReportPage';

import './VulnReportingPage.css';

function VulnReportingPage() {
    const { hasReadWriteAccess, hasReadAccess } = usePermissions();

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
            <Route exact path={vulnerabilityReportingPath} component={VulnReportsPage} />
            {canReadWriteReports && (
                <Route
                    exact
                    path={vulnerabilityReportingCreatePath}
                    component={CreateVulnReportPage}
                />
            )}
        </Switch>
    );
}

export default VulnReportingPage;
