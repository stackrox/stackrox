/* eslint-disable no-nested-ternary */
import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';

import CreateVulnReportPage from './ModifyVulnReport/CreateVulnReportPage';
import EditVulnReportPage from './ModifyVulnReport/EditVulnReportPage';
import CloneVulnReportPage from './ModifyVulnReport/CloneVulnReportPage';
import ViewVulnReportPage from './ViewVulnReport/ViewVulnReportPage';
import ConfigReportsTab from './VulnReports/ConfigReportsTab';
import OnDemandReportsTab from './VulnReports/OnDemandReportsTab';
import VulnReportingLayout from './VulnReports/VulnReportingLayout';

import './VulnReportingPage.css';

type PageActions = 'create' | 'edit' | 'clone';

function VulnReportingPage() {
    const { pageAction } = usePageAction<PageActions>();

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWriteAccessForReport =
        hasReadWriteAccess('WorkflowAdministration') &&
        hasReadAccess('Image') && // for vulnerabilities
        hasReadAccess('Integration'); // for notifiers

    // TODO: Modify routing for edge cases - https://github.com/stackrox/stackrox/pull/14873#discussion_r2042672432
    return (
        <Routes>
            <Route
                element={
                    pageAction === 'create' && hasWriteAccessForReport ? (
                        <CreateVulnReportPage />
                    ) : (
                        <VulnReportingLayout />
                    )
                }
            >
                <Route index element={<Navigate to="configuration" replace />} />
                <Route path="configuration" element={<ConfigReportsTab />} />
                <Route path="on-demand" element={<OnDemandReportsTab />} />
            </Route>
            <Route
                path="/configuration/:reportId"
                element={
                    pageAction === 'edit' && hasWriteAccessForReport ? (
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
