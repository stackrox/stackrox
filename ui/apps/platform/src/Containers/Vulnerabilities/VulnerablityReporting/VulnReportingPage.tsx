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
    const { pageAction } = usePageAction<PageActions>();

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWriteAccessForReport =
        hasReadWriteAccess('WorkflowAdministration') &&
        hasReadAccess('Image') && // for vulnerabilities
        hasReadAccess('Integration'); // for notifiers

    return (
        <>
            <TechPreviewBanner
                featureURL={vulnManagementReportsPath}
                featureName="Vulnerability Management (1.0) Reporting"
                routeKey="vulnerability-management/reports"
            />
            <Switch>
                <Route
                    exact
                    path={vulnerabilityReportsPath}
                    render={(props) => {
                        if (pageAction === 'create' && hasWriteAccessForReport) {
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
                        if (pageAction === 'edit' && hasWriteAccessForReport) {
                            return <EditVulnReportPage {...props} />;
                        }
                        if (pageAction === 'clone' && hasWriteAccessForReport) {
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
