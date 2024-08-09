import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportsPath } from 'routePaths';

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
        <Switch>
            <Route
                exact
                path={vulnerabilityReportsPath}
                // eslint-disable-next-line react/no-children-prop
                children={() => {
                    if (pageAction === 'create' && hasWriteAccessForReport) {
                        return <CreateVulnReportPage />;
                    }
                    if (pageAction === undefined) {
                        return <VulnReportsPage />;
                    }
                    return <Redirect to={vulnerabilityReportsPath} />;
                }}
            />
            <Route
                exact
                path={vulnerabilityReportPath}
                // eslint-disable-next-line react/no-children-prop
                children={() => {
                    if (pageAction === 'edit' && hasWriteAccessForReport) {
                        return <EditVulnReportPage />;
                    }
                    if (pageAction === 'clone' && hasWriteAccessForReport) {
                        return <CloneVulnReportPage />;
                    }
                    return <ViewVulnReportPage />;
                }}
            />
        </Switch>
    );
}

export default VulnReportingPage;
