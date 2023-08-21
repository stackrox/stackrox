import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import usePageAction from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';
import { vulnManagementReportsPath, vulnerabilityReportsPath } from 'routePaths';

import TechPreviewBanner from 'Components/TechPreviewBanner';
import VulnReportsPage from './VulnReports/VulnReportsPage';
import CreateVulnReportPage from './ModifyVulnReport/CreateVulnReportPage';
import EditVulnReportPage from './ModifyVulnReport/EditVulnReportPage';
import CloneVulnReportPage from './ModifyVulnReport/CloneVulnReportPage';
import ViewVulnReportPage from './ViewVulnReport/ViewVulnReportPage';

import { vulnerabilityReportPath } from './pathsForVulnerabilityReporting';

import './VulnReportingPage.css';

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
        <>
            <TechPreviewBanner
                featureURL={vulnManagementReportsPath}
                featureName="Vulnerability Management (1.0) Reporting"
            />
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
                <Route
                    exact
                    path={vulnerabilityReportPath}
                    render={(props) => {
                        if (pageAction === 'edit' && canReadWriteReports) {
                            return <EditVulnReportPage {...props} />;
                        }
                        if (pageAction === 'clone' && canReadWriteReports) {
                            return <CloneVulnReportPage {...props} />;
                        }
                        return <ViewVulnReportPage {...props} />;
                    }}
                />
            </Switch>
        </>
    );
}

export default VulnReportingPage;
