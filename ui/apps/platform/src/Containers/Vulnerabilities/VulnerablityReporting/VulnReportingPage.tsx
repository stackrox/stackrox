/* eslint-disable no-nested-ternary */
import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportsPath } from 'routePaths';

import VulnReportsPage from './VulnReports/VulnReportsPage';
import CreateVulnReportPage from './ModifyVulnReport/CreateVulnReportPage';
import EditVulnReportPage from './ModifyVulnReport/EditVulnReportPage';
import CloneVulnReportPage from './ModifyVulnReport/CloneVulnReportPage';
import ViewVulnReportPage from './ViewVulnReport/ViewVulnReportPage';

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
        <Routes>
            <Route
                index
                element={
                    pageAction === 'create' && hasWriteAccessForReport ? (
                        <CreateVulnReportPage />
                    ) : !pageAction ? (
                        <VulnReportsPage />
                    ) : (
                        <Navigate to={vulnerabilityReportsPath} replace />
                    )
                }
            />
            <Route
                path=":reportId"
                element={
                    pageAction === 'create' && hasWriteAccessForReport ? (
                        <EditVulnReportPage />
                    ) : pageAction === 'clone' && hasWriteAccessForReport ? (
                        <CloneVulnReportPage />
                    ) : (
                        <ViewVulnReportPage />
                    )
                }
            />
        </Routes>
    );
}

export default VulnReportingPage;
