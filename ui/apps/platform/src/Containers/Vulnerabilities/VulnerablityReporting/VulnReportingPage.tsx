import { Navigate, Route, Routes } from 'react-router-dom-v5-compat';

import PageNotFound from 'Components/PageNotFound'; // NofFoundPage from Body.tsx file would be even better
import type { ReportPageAction } from 'Components/Reports/reports.types';
import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';

import ImageVulnerabilityReportWizardPage from '../Reports/ImageVulnerabilityReports/Wizard/ImageVulnerabilityReportWizardPage';

import ViewVulnReportPage from './ViewVulnReport/ViewVulnReportPage';
import VulnReportingLayout from './VulnReports/VulnReportingLayout';

import './VulnReportingPage.css';

function VulnReportingPage() {
    const { pageAction } = usePageAction<ReportPageAction>();

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const isReportConfigurationEnabled = hasReadAccess('WorkflowAdministration');
    const hasWriteAccessForReport =
        hasReadWriteAccess('WorkflowAdministration') && hasReadAccess('Integration'); // for notifiers

    // TODO: Modify routing for edge cases - https://github.com/stackrox/stackrox/pull/14873#discussion_r2042672432
    return (
        <Routes>
            <Route
                path="/"
                element={
                    <Navigate
                        to={isReportConfigurationEnabled ? 'configurations' : 'view-based-jobs'}
                        replace
                    />
                }
            />
            {isReportConfigurationEnabled && (
                <Route
                    path="/configurations"
                    element={
                        (pageAction === 'create' || pageAction === 'createFromFilters') &&
                        hasWriteAccessForReport ? (
                            <ImageVulnerabilityReportWizardPage pageAction={pageAction} />
                        ) : (
                            <VulnReportingLayout />
                        )
                    }
                />
            )}
            {isReportConfigurationEnabled && (
                <Route
                    path="/configurations/:reportId"
                    element={
                        (pageAction === 'clone' || pageAction === 'edit') &&
                        hasWriteAccessForReport ? (
                            <ImageVulnerabilityReportWizardPage pageAction={pageAction} />
                        ) : (
                            <ViewVulnReportPage />
                        )
                    }
                />
            )}
            <Route path="/view-based-jobs" element={<VulnReportingLayout />} />
            <Route path="*" element={<PageNotFound />} />
        </Routes>
    );
}

export default VulnReportingPage;
