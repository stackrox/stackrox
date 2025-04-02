/* eslint-disable no-nested-ternary */
import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';

import { vulnerabilityConfigurationReportsPath } from 'routePaths';
import CreateVulnReportPage from './ModifyVulnReport/CreateVulnReportPage';
import EditVulnReportPage from './ModifyVulnReport/EditVulnReportPage';
import CloneVulnReportPage from './ModifyVulnReport/CloneVulnReportPage';
import ViewVulnReportPage from './ViewVulnReport/ViewVulnReportPage';

import './VulnReportingPage.css';
import ConfigReportsTab from './VulnReports/ConfigReportsTab';
import OnDemandReportsTab from './VulnReports/OnDemandReportsTab';
import ReportsLayout from './VulnReports/ReportsLayout';

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
                path={`${vulnerabilityConfigurationReportsPath}/:reportId`}
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
            <Route element={<ReportsLayout />}>
                <Route index element={<Navigate to="configuration" replace />} />
                <Route
                    path="configuration"
                    element={
                        pageAction === 'create' && hasWriteAccessForReport ? (
                            <CreateVulnReportPage />
                        ) : !pageAction ? (
                            <ConfigReportsTab />
                        ) : (
                            <Navigate to="configuration" replace />
                        )
                    }
                />
                <Route path="on-demand" element={<OnDemandReportsTab />} />
            </Route>
        </Routes>
    );
}

export default VulnReportingPage;

/*

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

*/
